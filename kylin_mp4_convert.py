# -*- coding: utf-8 -*-
# @Time    : 2025/03/15 10:05
# @Author  : 许木只
import sys
import os
import shutil
from PyQt5.QtWidgets import QApplication, QMainWindow, QLabel, QVBoxLayout, QWidget, QProgressBar, QSplitter, QMessageBox
from PyQt5.QtCore import Qt, QThread, pyqtSignal, QUrl
from PyQt5.QtGui import QDragEnterEvent, QDropEvent
from PyQt5.QtMultimedia import QMediaPlayer, QMediaContent
from PyQt5.QtMultimediaWidgets import QVideoWidget
import subprocess
import tempfile

class DropArea(QLabel):
    def __init__(self):
        super().__init__()
        self.setAlignment(Qt.AlignCenter)
        self.setText('>>将播放器打不开的公安视频文件拖放到这个框内进行转换<<\n生成的文件会保存在原目录')
        self.setStyleSheet(
            'QLabel{border: 2px dashed #aaa; border-radius: 5px; padding: 20px; background: #f0f0f0}'
        )
        self.setAcceptDrops(True)
        self.setMinimumSize(300, 200)

    def dragEnterEvent(self, event: QDragEnterEvent):
        if event.mimeData().hasUrls():
            for url in event.mimeData().urls():
                file_path = url.toLocalFile()
                if os.path.isfile(file_path):
                    event.acceptProposedAction()
                    return
            event.ignore()
        else:
            event.ignore()

    def dropEvent(self, event: QDropEvent):
        files = []
        for url in event.mimeData().urls():
            file_path = url.toLocalFile()
            if os.path.isfile(file_path):
                files.append(file_path)
        if files:
            event.acceptProposedAction()
            # 遍历父组件层级直到找到MainWindow实例
            parent = self.parent()
            while parent and not isinstance(parent, MainWindow):
                parent = parent.parent()
            if parent and isinstance(parent, MainWindow):
                parent.process_files(files)
        else:
            event.ignore()

class ConvertThread(QThread):
    progress = pyqtSignal(int)
    finished = pyqtSignal(str)
    error = pyqtSignal(str)
    stream_ready = pyqtSignal(str)  # 新增信号，用于通知视频流准备就绪

    def __init__(self, input_file):
        super().__init__()
        self.input_file = input_file
        self.temp_file = None

    def parse_time(self, time_str):
        try:
            h, m, s = time_str.split(':')
            seconds = float(h) * 3600 + float(m) * 60 + float(s)
            return seconds
        except:
            return 0

    def run(self):
        try:
            # 检查ffmpeg是否可用
            try:
                subprocess.run(['ffmpeg', '-version'], stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True)
            except (subprocess.SubprocessError, FileNotFoundError):
                self.error.emit('错误：未找到ffmpeg。请确保已安装ffmpeg并添加到系统路径。')
                return
                
            file_name, file_extension = os.path.splitext(self.input_file)
            if file_extension.lower() in ['.wav', '.mp3', '.ogg', '.m4a', '.flac']:
                output_extension = '.mp3'
            else:
                output_extension = '.mp4'
            
            # 检查输出文件是否已存在
            final_output = file_name + output_extension
            if os.path.exists(final_output):
                # 创建备份文件夹
                backup_dir = os.path.join(os.path.dirname(final_output), 'backup')
                os.makedirs(backup_dir, exist_ok=True)
                backup_file = os.path.join(backup_dir, os.path.basename(final_output))
                # 如果备份文件已存在，添加数字后缀
                if os.path.exists(backup_file):
                    i = 1
                    while os.path.exists(f"{backup_file}.{i}"):
                        i += 1
                    backup_file = f"{backup_file}.{i}"
                # 移动到备份文件夹
                shutil.move(final_output, backup_file)
            
            # 创建临时文件
            self.temp_file = tempfile.NamedTemporaryFile(suffix=output_extension, delete=False, mode='wb')
            output_file = self.temp_file.name
            self.temp_file.close()  # 关闭文件以确保其他进程可以访问
            
            # 获取输入文件的总时长
            probe_process = subprocess.Popen(
                ['ffmpeg', '-i', self.input_file],
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE
            )
            _, probe_output = probe_process.communicate()
            duration = 0
            for line in probe_output.decode('utf-8', errors='replace').split('\n'):
                if 'Duration:' in line:
                    time_str = line.split('Duration: ')[1].split(',')[0].strip()
                    duration = self.parse_time(time_str)
                    break
            
            # 执行ffmpeg命令，使用MPEGTS格式以支持边转换边播放
            process = subprocess.Popen(
                ['ffmpeg', '-y', '-i', self.input_file,
                 '-c:v', 'libx264', '-preset', 'ultrafast',  # 使用快速编码
                 '-c:a', 'aac',  # 音频编码
                 '-f', 'mpegts',  # 使用MPEGTS格式，更适合流媒体播放
                 output_file],
                stdout=subprocess.PIPE,
                stderr=subprocess.PIPE,
                universal_newlines=True,
                bufsize=1
            )
            
            # 等待一小段时间确保文件开始写入
            self.msleep(500)
            
            # 发送临时文件路径，开始播放
            self.stream_ready.emit(output_file)
            
            # 实时读取输出并更新进度
            while True:
                line = process.stderr.readline()
                if not line:
                    break
                if 'time=' in line:
                    time_str = line.split('time=')[1].split()[0].strip()
                    current_time = self.parse_time(time_str)
                    if duration > 0:
                        progress = int((current_time / duration) * 100)
                        self.progress.emit(progress)
            
            process.wait()
            
            if process.returncode == 0:
                self.progress.emit(100)
                os.rename(output_file, final_output)
                self.finished.emit(f'转换完成：{final_output}')
            else:
                error_output = process.stderr.read() if hasattr(process.stderr, 'read') else '未知错误'
                self.error.emit(f'转换失败：{error_output}')
        except Exception as e:
            self.error.emit(f'处理错误：{str(e)}')
        finally:
            # 清理临时文件
            if self.temp_file and os.path.exists(self.temp_file.name):
                try:
                    os.unlink(self.temp_file.name)
                except Exception as e:
                    print(f"清理临时文件失败: {e}")
                    pass

class MainWindow(QMainWindow):
    def __init__(self):
        super().__init__()
        self.setWindowTitle('温岭检察音视频解码工具')
        self.setMinimumSize(800, 600)
        self.setStyleSheet('QMainWindow { background-color: #f5f5f5; }')
        # 检查系统是否安装了ffmpeg
        self.check_ffmpeg()

        # 创建主窗口部件和布局
        self.central_widget = QWidget()
        self.setCentralWidget(self.central_widget)
        self.layout = QVBoxLayout(self.central_widget)
        self.layout.setContentsMargins(20, 20, 20, 20)
        self.layout.setSpacing(15)

        # 创建分割器
        self.splitter = QSplitter(Qt.Vertical)
        self.layout.addWidget(self.splitter)

        # 创建视频播放器
        self.video_widget = QVideoWidget()
        self.video_widget.setMinimumHeight(300)
        self.media_player = QMediaPlayer()
        self.media_player.setVideoOutput(self.video_widget)
        self.splitter.addWidget(self.video_widget)

        # 创建底部控制区域
        self.bottom_widget = QWidget()
        self.bottom_layout = QVBoxLayout(self.bottom_widget)
        self.splitter.addWidget(self.bottom_widget)

        # 创建拖放区域
        self.drop_area = DropArea()
        self.bottom_layout.addWidget(self.drop_area)

        # 创建进度条
        self.progress_bar = QProgressBar()
        self.progress_bar.setAlignment(Qt.AlignCenter)
        self.progress_bar.setStyleSheet("""
            QProgressBar {
                border: 1px solid #bbb;
                border-radius: 5px;
                text-align: center;
                height: 20px;
                background: #f0f0f0;
            }
            QProgressBar::chunk {
                background-color: #4CAF50;
                border-radius: 5px;
            }
        """)
        self.bottom_layout.addWidget(self.progress_bar)

        # 创建状态标签
        self.status_label = QLabel()
        self.status_label.setAlignment(Qt.AlignCenter)
        self.status_label.setStyleSheet('QLabel { color: #666; font-size: 13px; }')
        self.bottom_layout.addWidget(self.status_label)

        # 创建系统版本标签
        self.version_label = QLabel('技术支持：许钦滔\nArm64麒麟系统测试版')
        self.version_label.setAlignment(Qt.AlignCenter)
        self.version_label.setStyleSheet('QLabel { color: #999; font-size: 12px; padding: 10px; }')
        self.bottom_layout.addWidget(self.version_label)

        self.threads = []
        
    def check_ffmpeg(self):
        """检查系统是否安装了ffmpeg"""
        try:
            subprocess.run(['ffmpeg', '-version'], stdout=subprocess.PIPE, stderr=subprocess.PIPE, check=True)
        except (subprocess.SubprocessError, FileNotFoundError):
            QMessageBox.critical(self, '错误', 'ffmpeg未安装或未添加到系统路径。\n请安装ffmpeg后再使用本工具。')
            # 不立即退出，让用户看到错误信息

    def process_files(self, files):
        # 清理已完成的线程
        self.threads = [thread for thread in self.threads if thread.isRunning()]
        
        for file_path in files:
            # 检查文件是否存在且可读
            if not os.path.isfile(file_path) or not os.access(file_path, os.R_OK):
                self.status_label.setText(f'错误：无法访问文件 {file_path}')
                self.status_label.setStyleSheet('color: red')
                continue
                
            # 显示正在处理的文件名
            self.status_label.setText(f'正在处理：{os.path.basename(file_path)}')
            self.status_label.setStyleSheet('color: blue')
            
            thread = ConvertThread(file_path)
            thread.finished.connect(self.on_conversion_finished)
            thread.error.connect(self.on_conversion_error)
            thread.progress.connect(self.on_progress_update)
            thread.stream_ready.connect(self.on_stream_ready)
            thread.start()
            self.threads.append(thread)
            self.progress_bar.show()
            self.progress_bar.setValue(0)

    def on_stream_ready(self, file_path):
        try:
            # 停止当前播放
            if self.media_player.state() != QMediaPlayer.StoppedState:
                self.media_player.stop()
                
            self.media_player.setMedia(QMediaContent(QUrl.fromLocalFile(file_path)))
            self.media_player.error.connect(lambda: self.on_player_error(self.media_player.errorString()))
            self.media_player.stateChanged.connect(self.on_player_state_changed)
            self.media_player.play()
        except Exception as e:
            self.status_label.setText(f'播放错误：{str(e)}')
            self.status_label.setStyleSheet('color: red')
            
    def on_player_state_changed(self, state):
        """处理播放器状态变化"""
        if state == QMediaPlayer.StoppedState:
            # 播放结束时重新播放
            if self.media_player.position() >= self.media_player.duration() and self.media_player.duration() > 0:
                self.media_player.setPosition(0)
                self.media_player.play()

    def on_player_error(self, error_string):
        self.status_label.setText(f'播放器错误：{error_string}')
        self.status_label.setStyleSheet('color: red')

    def on_conversion_finished(self, message):
        self.status_label.setText(message)
        self.status_label.setStyleSheet('color: green')

    def on_conversion_error(self, error_message):
        self.status_label.setText(error_message)
        self.status_label.setStyleSheet('color: red')

    def on_progress_update(self, value):
        self.progress_bar.setValue(value)

def main():
    app = QApplication(sys.argv)
    app.setStyle('Fusion')  # 使用Fusion风格，在不同平台上有更一致的外观
    window = MainWindow()
    window.show()
    sys.exit(app.exec_())

if __name__ == '__main__':
    main()