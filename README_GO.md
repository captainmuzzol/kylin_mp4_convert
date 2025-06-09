# 音视频解码工具 - Go重构版

## 📝 简介

这是原Python版本音视频解码工具的Go语言重构版本。该工具专为专网环境设计，主要用于解决特定格式的公安视频文件无法在普通播放器中播放的问题。Go版本具有更好的性能、更小的内存占用和更简单的部署。

## 🆕 Go版本特性

- ✅ **跨平台支持**：支持Windows、macOS、Linux（包括麒麟系统）
- ✅ **单文件部署**：编译后生成单个可执行文件，无需安装运行时
- ✅ **更好的性能**：Go语言的高效并发处理
- ✅ **现代化UI**：使用Fyne框架构建的现代化界面
- ✅ **实时进度监控**：精确的转换进度显示
- ✅ **智能文件备份**：自动备份同名文件到backup目录

## 🔧 安装要求

### 系统要求
- 支持麒麟系统（Arm64架构）
- Windows 10/11
- macOS 10.14+
- Linux发行版（Ubuntu、CentOS等）

### 依赖项
- **Go 1.21+**（仅开发时需要）
- **FFmpeg**（必须安装并添加到系统路径）

### 安装FFmpeg

#### 麒麟系统/Ubuntu/Debian
```bash
sudo apt-get update
sudo apt-get install ffmpeg
```

#### CentOS/RHEL
```bash
sudo yum install epel-release
sudo yum install ffmpeg
```

#### macOS
```bash
brew install ffmpeg
```

#### Windows
1. 从 [FFmpeg官网](https://ffmpeg.org/download.html) 下载Windows版本
2. 解压到任意目录（如 `C:\ffmpeg`）
3. 将 `C:\ffmpeg\bin` 添加到系统PATH环境变量

## 🚀 快速开始

### 方式一：使用预编译版本（推荐）

1. 从Releases页面下载对应平台的可执行文件
2. 双击运行即可使用

### 方式二：从源码编译

1. **安装Go环境**
   ```bash
   # 下载并安装Go 1.21+
   # https://golang.org/dl/
   ```

2. **克隆项目**
   ```bash
   git clone <repository-url>
   cd kylin_mp4_convert
   ```

3. **安装依赖**
   ```bash
   go mod tidy
   ```

4. **编译运行**
   ```bash
   # 直接运行
   go run .
   
   # 或编译后运行
   go build -o mp4_converter
   ./mp4_converter
   ```

5. **交叉编译**（可选）
   ```bash
   # 编译Linux版本（在其他平台上）
   GOOS=linux GOARCH=amd64 go build -o mp4_converter_linux
   
   # 编译Windows版本
   GOOS=windows GOARCH=amd64 go build -o mp4_converter.exe
   
   # 编译macOS版本
   GOOS=darwin GOARCH=amd64 go build -o mp4_converter_mac
   
   # 编译ARM64版本（适用于麒麟系统）
   GOOS=linux GOARCH=arm64 go build -o mp4_converter_arm64
   ```

## 🎯 使用方法

1. **启动应用程序**
   - 双击可执行文件或在终端中运行

2. **选择文件**
   - 点击"选择文件"按钮选择需要转换的视频文件
   - 支持的格式：MP4、AVI、MOV、MKV、FLV、WMV、WebM、M4V、3GP等
   - 音频格式：WAV、MP3、OGG、M4A、FLAC、AAC等

3. **开始转换**
   - 选择文件后自动开始转换
   - 实时显示转换进度
   - 转换完成后会弹出提示

4. **查看结果**
   - 转换后的文件保存在原文件相同目录
   - 视频文件转换为MP4格式
   - 音频文件转换为MP3格式

## 📋 注意事项

- ✅ **文件备份**：如果目标目录已存在同名文件，原文件会自动备份到`backup`文件夹
- ✅ **路径支持**：支持中文路径和文件名
- ✅ **格式检测**：自动检测文件类型并选择合适的输出格式
- ⚠️ **FFmpeg依赖**：确保系统已正确安装FFmpeg
- ⚠️ **磁盘空间**：确保有足够的磁盘空间存储转换后的文件

## 🔍 故障排除

### 常见问题

1. **"ffmpeg未安装或未添加到系统路径"错误**
   - 确保已安装FFmpeg
   - 检查FFmpeg是否添加到系统PATH
   - 在终端中运行 `ffmpeg -version` 验证安装

2. **转换失败**
   - 检查输入文件是否损坏
   - 确保有足够的磁盘空间
   - 检查文件权限

3. **界面无法显示**
   - 确保系统支持图形界面
   - 在Linux上可能需要安装额外的图形库

### 日志调试

在终端中运行程序可以看到详细的错误信息：
```bash
./mp4_converter
```

## 🏗️ 项目结构

```
kylin_mp4_convert/
├── main.go              # 主程序和UI逻辑
├── converter.go         # FFmpeg转换逻辑
├── go.mod              # Go模块依赖
├── go.sum              # 依赖校验文件
├── README_GO.md        # Go版本说明文档
└── kylin_mp4_convert.py # 原Python版本（保留）
```

## 🤝 贡献

欢迎提交Issue和Pull Request来改进这个项目！

## 📄 许可证

本项目采用与原Python版本相同的许可证。

## 🔄 版本对比

| 特性 | Python版本 | Go版本 |
|------|------------|--------|
| 运行时依赖 | Python + PyQt5 | 仅需FFmpeg |
| 部署方式 | 需要Python环境 | 单文件部署 |
| 内存占用 | 较高 | 较低 |
| 启动速度 | 较慢 | 快速 |
| 跨平台 | 需要安装依赖 | 原生支持 |
| 界面框架 | PyQt5 | Fyne |
| 拖拽支持 | ✅ | 文件选择 |
| 实时预览 | ✅ | 计划中 |

---

**注意**：Go版本目前不支持实时预览功能，这个功能在后续版本中会添加。如果需要实时预览功能，请使用Python版本。