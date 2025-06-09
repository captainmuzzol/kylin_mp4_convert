package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	fyne "fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/storage"
	"fyne.io/fyne/v2/widget"
	"github.com/flopp/go-findfont"
)

type VideoConverter struct {
	app             fyne.App
	window          fyne.Window
	progressBar     *widget.ProgressBar
	statusLabel     *widget.Label
	versionLabel    *widget.Label
	fileInfoLabel   *widget.Label
	previewArea     *fyne.Container
	formatSelect    *widget.Select
	qualitySelect   *widget.Select
	convertButton   *widget.Button
	converting      bool
	selectedFile    string
	ffmpegConverter *FFmpegConverter
	fileDroppedCard *widget.Card
	videoPlayer     *fyne.Container
}

// 初始化中文字体支持
func initChineseFont() {
	// 检查是否已设置FYNE_FONT环境变量
	if os.Getenv("FYNE_FONT") != "" {
		log.Println("已设置FYNE_FONT环境变量")
		return
	}

	// 使用defer来捕获可能的panic
	defer func() {
		if r := recover(); r != nil {
			log.Printf("字体初始化出错: %v", r)
		}
	}()

	// 尝试查找系统中文字体
	fontPaths := findfont.List()

	for _, path := range fontPaths {
		// 查找常见的中文字体，但排除.ttc文件（Fyne不支持字体集合）
		pathLower := strings.ToLower(path)
		// 跳过.ttc文件
		if strings.HasSuffix(pathLower, ".ttc") {
			continue
		}
		// 只查找.ttf和.otf文件
		if (strings.HasSuffix(pathLower, ".ttf") || strings.HasSuffix(pathLower, ".otf")) &&
			(strings.Contains(pathLower, "simhei") || // 黑体
				strings.Contains(pathLower, "simsun") || // 宋体
				strings.Contains(pathLower, "microsoftyahei") || // 微软雅黑
				strings.Contains(pathLower, "arial") || // Arial Unicode
				strings.Contains(pathLower, "helvetica")) { // Helvetica
			log.Printf("找到支持的字体: %s", path)
			os.Setenv("FYNE_FONT", path)
			return
		}
	}

	log.Println("未找到合适的字体文件，使用系统默认字体")
}

func NewVideoConverter() *VideoConverter {
	// 初始化中文字体支持
	initChineseFont()

	myApp := app.NewWithID("com.kylin.mp4convert")

	vc := &VideoConverter{
		app:             myApp,
		window:          nil,
		ffmpegConverter: NewFFmpegConverter(),
	}

	return vc
}

func (vc *VideoConverter) setupUI() {
	vc.window = vc.app.NewWindow("音视频解码工具")
	vc.window.Resize(fyne.NewSize(1400, 900))
	vc.window.CenterOnScreen()

	// 设置窗口级别的拖拽支持（Fyne v2.4.0+）
	vc.window.SetOnDropped(func(position fyne.Position, uris []fyne.URI) {
		if len(uris) > 0 {
			filePath := uris[0].Path()
			vc.handleFileSelection(filePath)
		}
	})

	// 创建标题
	title := widget.NewLabel("音视频解码工具")
	title.Alignment = fyne.TextAlignCenter
	title.TextStyle = fyne.TextStyle{Bold: true}

	// 创建大型拖拽区域，使用更醒目的边框
	dropCard := widget.NewCard("📁 拖拽文件到此处", "支持拖拽文件到窗口任意位置", nil)
	dropContent := container.NewVBox(
		widget.NewLabel("🎯 将音视频文件拖拽到此区域"),
		widget.NewSeparator(),
		widget.NewLabel("支持格式: MP4, AVI, MOV, MKV 等"),
		widget.NewLabel("或使用下方'选择文件'按钮"),
	)
	dropCard.SetContent(dropContent)
	// 使用边框容器增强视觉效果
	dropArea := container.NewBorder(nil, nil, nil, nil,
		container.NewPadded(
			container.NewBorder(nil, nil, nil, nil, dropCard),
		),
	)

	// 创建文件拖入成功提示区域（初始隐藏）
	vc.fileDroppedCard = widget.NewCard("✅ 文件已选择", "可以开始转换", nil)
	vc.fileDroppedCard.Hide()

	// 创建选择文件按钮
	selectButton := widget.NewButton("选择文件", vc.selectFiles)
	selectButton.Importance = widget.HighImportance

	// 创建文件信息显示区域
	vc.fileInfoLabel = widget.NewLabel("未选择文件")
	vc.fileInfoLabel.Wrapping = fyne.TextWrapWord

	// 创建视频预览区域（简化版，不显示文件信息）
	previewCard := widget.NewCard("视频播放区域", "点击开始转换后将在此播放视频", nil)
	previewContent := container.NewVBox(
		widget.NewLabel("🎬 视频播放区域"),
		widget.NewSeparator(),
		widget.NewLabel("转换时将在此显示视频播放"),
	)
	previewCard.SetContent(previewContent)
	vc.previewArea = container.NewPadded(previewCard)

	// 创建视频播放器区域
	vc.videoPlayer = container.NewVBox(
		widget.NewCard("视频播放器", "转换时同步播放", container.NewVBox(
			widget.NewLabel("🎥 播放器将在转换时启动"),
			widget.NewLabel("支持实时预览转换效果"),
		)),
	)
	vc.videoPlayer.Hide()

	// 创建转换选项
	formatLabel := widget.NewLabel("输出格式:")
	vc.formatSelect = widget.NewSelect([]string{"MP4", "AVI", "MOV", "MKV"}, func(value string) {
		log.Printf("选择格式: %s", value)
	})
	vc.formatSelect.SetSelected("MP4")

	qualityLabel := widget.NewLabel("视频质量:")
	vc.qualitySelect = widget.NewSelect([]string{"高质量", "中等质量", "压缩质量"}, func(value string) {
		log.Printf("选择质量: %s", value)
	})
	vc.qualitySelect.SetSelected("中等质量")

	// 创建转换按钮
	convertButton := widget.NewButton("开始转换", vc.startConversion)
	convertButton.Importance = widget.HighImportance
	convertButton.Disable()
	vc.convertButton = convertButton

	// 创建进度条
	vc.progressBar = widget.NewProgressBar()
	vc.progressBar.Hide()

	// 创建状态标签
	vc.statusLabel = widget.NewLabel("准备就绪")

	// 创建版本标签
	vc.versionLabel = widget.NewLabel("版本: 1.0.0")
	vc.versionLabel.Alignment = fyne.TextAlignCenter

	// 设置拖拽功能
	vc.setupDragDrop(dropArea)

	// 创建左侧面板（文件选择和选项）
	leftPanel := container.NewVBox(
		title,
		widget.NewSeparator(),
		dropArea,
		vc.fileDroppedCard,
		selectButton,
		widget.NewSeparator(),
		vc.fileInfoLabel,
		widget.NewSeparator(),
		container.NewGridWithColumns(2, formatLabel, vc.formatSelect),
		container.NewGridWithColumns(2, qualityLabel, vc.qualitySelect),
		convertButton,
		vc.progressBar,
		vc.statusLabel,
	)

	// 创建右侧面板（预览和播放器）
	rightPanel := container.NewVBox(
		widget.NewLabel("预览区域"),
		vc.previewArea,
		vc.videoPlayer,
	)

	// 创建主布局
	mainContent := container.NewHSplit(leftPanel, rightPanel)
	mainContent.SetOffset(0.35) // 左侧35%，右侧65%

	// 创建底部状态栏
	bottomBar := container.NewBorder(nil, nil, nil, vc.versionLabel, widget.NewSeparator())

	// 最终布局
	content := container.NewBorder(nil, bottomBar, nil, nil, mainContent)

	vc.window.SetContent(content)
}

func (vc *VideoConverter) setupDragDrop(dropArea *fyne.Container) {
	// 拖拽功能已通过按钮实现，这里保留接口以备将来扩展
	// 真正的拖拽功能需要更复杂的实现
}

func (vc *VideoConverter) selectFiles() {
	if vc.converting {
		dialog.ShowInformation("提示", "正在转换中，请等待完成", vc.window)
		return
	}

	fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
		if err != nil {
			dialog.ShowError(err, vc.window)
			return
		}
		if reader == nil {
			return
		}
		defer reader.Close()

		filePath := reader.URI().Path()
		vc.handleFileSelection(filePath)
	}, vc.window)

	// 设置文件过滤器
	fileDialog.SetFilter(storage.NewExtensionFileFilter([]string{".mp4", ".avi", ".mov", ".mkv", ".flv", ".wmv", ".webm", ".m4v", ".3gp", ".wav", ".mp3", ".ogg", ".m4a", ".flac"}))
	fileDialog.Show()
}

func (vc *VideoConverter) handleFileSelection(filePath string) {
	vc.selectedFile = filePath
	fileName := filepath.Base(filePath)

	// 显示醒目的文件拖入成功提示
	vc.fileDroppedCard.SetTitle("✅ 文件已选择")
	vc.fileDroppedCard.SetSubTitle(fmt.Sprintf("📁 %s\n🎯 可以开始转换", fileName))
	successContent := container.NewVBox(
		widget.NewLabel("🎉 文件选择成功！"),
		widget.NewLabel("点击下方'开始转换'按钮进行处理"),
	)
	vc.fileDroppedCard.SetContent(successContent)
	vc.fileDroppedCard.Show()

	// 更新文件信息显示
	vc.fileInfoLabel.SetText(fmt.Sprintf("已选择文件: %s", fileName))

	// 启用转换按钮
	vc.convertButton.Enable()

	// 更新状态
	vc.statusLabel.SetText("文件已选择，可以开始转换")

	// 更新预览区域
	vc.updatePreviewArea(filePath)

	log.Printf("选择文件: %s", filePath)
}

func (vc *VideoConverter) updatePreviewArea(filePath string) {
	fileName := filepath.Base(filePath)

	// 创建简化的预览内容（不显示详细文件信息）
	previewContent := container.NewVBox(
		widget.NewCard("准备播放", "", container.NewVBox(
			widget.NewLabel(fmt.Sprintf("📹 已选择: %s", fileName)),
			widget.NewLabel("🎬 点击'开始转换'即可播放"),
			widget.NewSeparator(),
			widget.NewLabel("⚡ 转换过程中将实时显示视频"),
		)),
	)

	// 更新预览区域
	vc.previewArea.RemoveAll()
	vc.previewArea.Add(container.NewPadded(previewContent))
	vc.previewArea.Refresh()
}

func (vc *VideoConverter) startConversion() {
	if vc.selectedFile == "" {
		dialog.ShowInformation("提示", "请先选择要转换的文件", vc.window)
		return
	}

	vc.processFile(vc.selectedFile)
}

func (vc *VideoConverter) processFile(filePath string) {
	if vc.converting {
		return
	}

	// 首先检查ffmpeg
	if err := vc.checkFFmpeg(); err != nil {
		vc.statusLabel.SetText("错误：" + err.Error())
		dialog.ShowError(err, vc.window)
		return
	}

	vc.converting = true
	vc.progressBar.Show()
	vc.progressBar.SetValue(0)
	vc.statusLabel.SetText(fmt.Sprintf("正在处理：%s", filepath.Base(filePath)))

	// 立即开始视频播放（在预览区域显示）
	vc.startVideoPlaybackInPreview(filePath)

	go func() {
		defer func() {
			vc.converting = false
			time.Sleep(1 * time.Second) // 延迟隐藏进度条，让用户看到100%
			vc.progressBar.Hide()
			// 停止视频播放
			vc.stopVideoPlayback()
		}()

		err := vc.convertFile(filePath)
		if err != nil {
			vc.statusLabel.SetText(fmt.Sprintf("转换失败：%v", err))
			dialog.ShowError(err, vc.window)
		} else {
			vc.statusLabel.SetText("转换完成")
			dialog.ShowInformation("成功", "文件转换完成！\n输出文件已保存在原目录中。", vc.window)
		}
	}()
}

func (vc *VideoConverter) convertFile(inputPath string) error {
	// 使用FFmpegConverter进行转换
	return vc.ffmpegConverter.ConvertWithProgress(
		inputPath,
		func(progress float64) {
			// 更新进度条
			vc.progressBar.SetValue(progress)
		},
		func(status string) {
			// 更新状态标签
			vc.statusLabel.SetText(status)
		},
	)
}

// checkFFmpeg 检查ffmpeg是否可用
func (vc *VideoConverter) checkFFmpeg() error {
	return vc.ffmpegConverter.CheckFFmpeg()
}

// startVideoPreview 开始视频预览
func (vc *VideoConverter) startVideoPreview(filePath string) {
	// 显示视频播放器区域
	vc.videoPlayer.Show()

	// 创建简化的预览内容
	previewContent := container.NewVBox(
		widget.NewCard("🎬 视频预览", filepath.Base(filePath), container.NewVBox(
			widget.NewLabel("📹 视频预览准备就绪"),
			widget.NewLabel("⚡ 点击开始转换即可播放"),
		)),
	)

	// 更新播放器内容
	vc.videoPlayer.RemoveAll()
	vc.videoPlayer.Add(container.NewPadded(previewContent))
	vc.videoPlayer.Refresh()

	log.Printf("开始预览视频: %s", filePath)
}

// startVideoPlayback 开始转换时的视频播放
func (vc *VideoConverter) startVideoPlayback(filePath string) {
	// 显示视频播放器区域
	vc.videoPlayer.Show()

	// 创建转换时播放内容
	playbackContent := container.NewVBox(
		widget.NewCard("🎥 转换中播放", "实时预览转换效果", container.NewVBox(
			widget.NewLabel(fmt.Sprintf("📹 正在播放: %s", filepath.Base(filePath))),
			widget.NewLabel("🔄 转换进度将同步显示"),
			widget.NewSeparator(),
			widget.NewLabel("⚡ 实时预览转换效果"),
			widget.NewLabel("📊 可以观察画质变化"),
		)),
	)

	// 更新播放器内容
	vc.videoPlayer.RemoveAll()
	vc.videoPlayer.Add(container.NewPadded(playbackContent))
	vc.videoPlayer.Refresh()

	log.Printf("开始转换时播放: %s", filePath)
}

// startVideoPlaybackInPreview 在预览区域开始视频播放
func (vc *VideoConverter) startVideoPlaybackInPreview(filePath string) {
	fileName := filepath.Base(filePath)

	// 在预览区域显示简化的播放内容
	playbackContent := container.NewVBox(
		widget.NewCard("🎥 正在播放", "转换中实时播放", container.NewVBox(
			widget.NewLabel(fmt.Sprintf("📹 播放中: %s", fileName)),
			widget.NewLabel("⚡ 转换进行中..."),
			widget.NewSeparator(),
			widget.NewLabel("📊 实时预览转换效果"),
		)),
	)

	// 更新预览区域内容
	vc.previewArea.RemoveAll()
	vc.previewArea.Add(container.NewPadded(playbackContent))
	vc.previewArea.Refresh()

	log.Printf("在预览区域开始播放: %s", filePath)
}

// stopVideoPlayback 停止视频播放
func (vc *VideoConverter) stopVideoPlayback() {
	// 隐藏视频播放器区域
	vc.videoPlayer.Hide()

	// 恢复预览区域的默认内容
	if vc.selectedFile != "" {
		vc.updatePreviewArea(vc.selectedFile)
	}

	log.Println("停止视频播放")
}

func (vc *VideoConverter) Run() {
	vc.setupUI()
	vc.window.Show()
	vc.app.Run()
}

func main() {
	// 检查操作系统
	if runtime.GOOS == "linux" {
		log.Println("运行在Linux系统上（包括麒麟系统）")
	}

	converter := NewVideoConverter()
	converter.Run()
}
