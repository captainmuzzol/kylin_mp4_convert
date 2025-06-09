package main

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// FFmpegConverter 处理视频转换的结构体
type FFmpegConverter struct {
	progressCallback func(float64)
	statusCallback   func(string)
}

// NewFFmpegConverter 创建新的转换器实例
func NewFFmpegConverter() *FFmpegConverter {
	return &FFmpegConverter{}
}

// SetProgressCallback 设置进度回调函数
func (fc *FFmpegConverter) SetProgressCallback(callback func(float64)) {
	fc.progressCallback = callback
}

// SetStatusCallback 设置状态回调函数
func (fc *FFmpegConverter) SetStatusCallback(callback func(string)) {
	fc.statusCallback = callback
}

// CheckFFmpeg 检查系统是否安装了ffmpeg
func (fc *FFmpegConverter) CheckFFmpeg() error {
	cmd := exec.Command("ffmpeg", "-version")
	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("ffmpeg未安装或未添加到系统路径")
	}
	return nil
}

// GetOutputPath 根据输入文件路径生成输出文件路径
func (fc *FFmpegConverter) GetOutputPath(inputPath string) string {
	ext := strings.ToLower(filepath.Ext(inputPath))
	dir := filepath.Dir(inputPath)
	name := strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath))

	// 根据文件类型确定输出格式
	audioExts := []string{".wav", ".mp3", ".ogg", ".m4a", ".flac", ".aac"}
	for _, audioExt := range audioExts {
		if ext == audioExt {
			return filepath.Join(dir, name+".mp3")
		}
	}
	return filepath.Join(dir, name+".mp4")
}

// BackupExistingFile 备份已存在的文件
func (fc *FFmpegConverter) BackupExistingFile(outputPath string) error {
	if _, err := os.Stat(outputPath); os.IsNotExist(err) {
		return nil // 文件不存在，无需备份
	}

	backupDir := filepath.Join(filepath.Dir(outputPath), "backup")
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("创建备份目录失败：%v", err)
	}

	backupPath := filepath.Join(backupDir, filepath.Base(outputPath))

	// 如果备份文件已存在，添加数字后缀
	i := 1
	for {
		if _, err := os.Stat(backupPath); os.IsNotExist(err) {
			break
		}
		backupPath = filepath.Join(backupDir, fmt.Sprintf("%s.%d", filepath.Base(outputPath), i))
		i++
	}

	if fc.statusCallback != nil {
		fc.statusCallback(fmt.Sprintf("备份原文件到：%s", backupPath))
	}

	return os.Rename(outputPath, backupPath)
}

// GetVideoDuration 获取视频时长
func (fc *FFmpegConverter) GetVideoDuration(inputPath string) (float64, error) {
	cmd := exec.Command("ffmpeg", "-i", inputPath)
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return 0, err
	}

	if err := cmd.Start(); err != nil {
		return 0, err
	}

	scanner := bufio.NewScanner(stderr)
	var duration float64

	for scanner.Scan() {
		line := scanner.Text()
		if strings.Contains(line, "Duration:") {
			// 解析时长信息，格式如：Duration: 00:01:30.45
			re := regexp.MustCompile(`Duration: (\d{2}):(\d{2}):(\d{2})\.(\d{2})`)
			matches := re.FindStringSubmatch(line)
			if len(matches) == 5 {
				hours, _ := strconv.Atoi(matches[1])
				minutes, _ := strconv.Atoi(matches[2])
				seconds, _ := strconv.Atoi(matches[3])
				milliseconds, _ := strconv.Atoi(matches[4])

				duration = float64(hours*3600+minutes*60+seconds) + float64(milliseconds)/100.0
				break
			}
		}
	}

	cmd.Wait()
	return duration, nil
}

// ConvertFile 转换文件
func (fc *FFmpegConverter) ConvertFile(inputPath, outputPath string) error {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 获取视频时长用于计算进度
	duration, err := fc.GetVideoDuration(inputPath)
	if err != nil {
		duration = 0 // 如果获取失败，设为0，进度将无法准确显示
	}

	// 构建ffmpeg命令
	args := []string{
		"-y",            // 覆盖输出文件
		"-i", inputPath, // 输入文件
		"-c:v", "libx264", // 视频编码器
		"-preset", "ultrafast", // 编码速度预设
		"-c:a", "aac", // 音频编码器
		"-progress", "pipe:2", // 输出进度到stderr
		outputPath, // 输出文件
	}

	cmd := exec.CommandContext(ctx, "ffmpeg", args...)

	// 获取stderr用于进度监控
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return fmt.Errorf("创建stderr管道失败：%v", err)
	}

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动ffmpeg失败：%v", err)
	}

	// 监控进度
	go fc.monitorProgress(stderr, duration)

	if err := cmd.Wait(); err != nil {
		return fmt.Errorf("ffmpeg执行失败：%v", err)
	}

	if fc.progressCallback != nil {
		fc.progressCallback(1.0) // 设置为100%
	}

	return nil
}

// monitorProgress 监控转换进度
func (fc *FFmpegConverter) monitorProgress(stderr io.ReadCloser, totalDuration float64) {
	scanner := bufio.NewScanner(stderr)
	for scanner.Scan() {
		line := scanner.Text()

		// 解析时间信息，格式如：out_time=00:01:30.450000
		if strings.HasPrefix(line, "out_time=") {
			timeStr := strings.TrimPrefix(line, "out_time=")
			currentTime := fc.parseTimeString(timeStr)

			if totalDuration > 0 && currentTime > 0 {
				progress := currentTime / totalDuration
				if progress > 1.0 {
					progress = 1.0
				}

				if fc.progressCallback != nil {
					fc.progressCallback(progress)
				}

				if fc.statusCallback != nil {
					fc.statusCallback(fmt.Sprintf("转换进度：%.1f%%", progress*100))
				}
			}
		}
	}
}

// parseTimeString 解析时间字符串，格式如：00:01:30.450000
func (fc *FFmpegConverter) parseTimeString(timeStr string) float64 {
	re := regexp.MustCompile(`(\d{2}):(\d{2}):(\d{2})\.(\d+)`)
	matches := re.FindStringSubmatch(timeStr)
	if len(matches) != 5 {
		return 0
	}

	hours, _ := strconv.Atoi(matches[1])
	minutes, _ := strconv.Atoi(matches[2])
	seconds, _ := strconv.Atoi(matches[3])
	microseconds, _ := strconv.Atoi(matches[4])

	// 将微秒转换为秒
	microsecondsFloat := float64(microseconds) / 1000000.0

	return float64(hours*3600+minutes*60+seconds) + microsecondsFloat
}

// ConvertWithProgress 带进度监控的转换函数
func (fc *FFmpegConverter) ConvertWithProgress(inputPath string, progressCallback func(float64), statusCallback func(string)) error {
	fc.SetProgressCallback(progressCallback)
	fc.SetStatusCallback(statusCallback)

	// 检查ffmpeg
	if err := fc.CheckFFmpeg(); err != nil {
		return err
	}

	// 生成输出路径
	outputPath := fc.GetOutputPath(inputPath)

	// 备份已存在的文件
	if err := fc.BackupExistingFile(outputPath); err != nil {
		return err
	}

	if statusCallback != nil {
		statusCallback(fmt.Sprintf("开始转换：%s", filepath.Base(inputPath)))
	}

	// 执行转换
	if err := fc.ConvertFile(inputPath, outputPath); err != nil {
		return err
	}

	if statusCallback != nil {
		statusCallback(fmt.Sprintf("转换完成：%s", outputPath))
	}

	return nil
}
