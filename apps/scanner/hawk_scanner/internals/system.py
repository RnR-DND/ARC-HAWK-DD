from rich.console import Console
import json
import requests
import argparse
import yaml
import re
import datetime
import os
import subprocess
import platform
import hashlib
from tinydb import TinyDB, Query
import numpy as np
import pytesseract
from PIL import Image, ImageEnhance
from docx import Document
from openpyxl import load_workbook
from concurrent.futures import ThreadPoolExecutor, as_completed
import PyPDF2
import patoolib
import tempfile
import shutil
import cv2
import tarfile
import pkg_resources
from concurrent.futures import ProcessPoolExecutor


data_sources = ['s3', 'mysql', 'redis', 'firebase', 'gcs', 'fs', 'postgresql',
            'mongodb', 'slack', 'couchdb', 'gdrive', 'gdrive_workspace', 'text']
data_sources_option = ['all'] + data_sources


def parse_args(args=None):
	try:
		from hawk_scanner.__version__ import VERSION
		version = VERSION
	except ImportError:
		try:
			version = pkg_resources.require("hawk_scanner")[0].version
		except Exception:
			version = "0.3.39"
	parser = argparse.ArgumentParser(description='🦅 A powerful scanner to scan your Filesystem, S3, MySQL, PostgreSQL, MongoDB, Redis, Google Cloud Storage and Firebase storage for PII and sensitive data.')
	parser.add_argument('command', nargs='?', choices=data_sources_option, help='Command to execute')
	parser.add_argument('--connection', action='store', help='YAML Connection file path')
	parser.add_argument('--config', action='store', help='JSON Scan config file path (from Go backend)')
	parser.add_argument('--connection-json', type=str, help='Connection details in JSON format, useful for passing connection info directly as CLI Input')
	parser.add_argument('--fingerprint', action='store', help='Override YAML fingerprint file path')
	parser.add_argument('--json', help='Save output to a json file')
	parser.add_argument('--csv', help='Save output to a csv file')
	parser.add_argument('--stdout', action='store_true', help='Print output to stdout in JSON format')
	parser.add_argument('--quiet', action='store_true', help='Print only the results')
	parser.add_argument('--debug', action='store_true', help='Enable debug mode')
	parser.add_argument('--no-write', action='store_true', help='Do not write previous alerts to file, this may flood you with duplicate alerts')
	parser.add_argument('--shutup', action='store_true', help='Suppress the Hawk Eye banner 🫣', default=False)
	parser.add_argument('--version', action='version', version='%(prog)s v' + version)
	parser.add_argument('--hawk-thuu', action='store_true', help="Delete all spitted files during testing phase forcefully")
	# NEW: Auto-ingestion support
	parser.add_argument('--ingest-url', type=str, help='Automatically POST scan results to backend API (e.g., http://localhost:8080/api/v1/ingest)')
	parser.add_argument('--ingest-retry', type=int, default=3, help='Number of retries for ingestion (default: 3)')
	parser.add_argument('--ingest-timeout', type=int, default=30, help='Timeout for ingestion request in seconds (default: 30)')
	return parser.parse_args(args)


console = Console()

def calculate_msg_hash(msg):
    return hashlib.sha256(msg.encode()).hexdigest()


def print_info(args, message):
    if not args.quiet:
        console.print(f"[yellow][INFO][/yellow] {str(message)}")


def print_debug(args, message):
    if args and type(args) == argparse.Namespace and args.debug and not args.quiet:
        try:
            console.print(f"[blue][DEBUG][/blue] {str(message)}")
        except Exception:
            pass


def print_error(args, message):
    if not args.quiet:
        console.print(f"[bold red]❌ {message}")


def print_success(args, message):
    if not args.quiet:
        console.print(f"[bold green]✅ {message}")


def get_file_owner(file_path):
    owner_name = ""

    # Determine the current operating system
    system = platform.system()

    if system == "Windows":
        try:
            # Run the 'dir /q' command and capture its output
            result = subprocess.check_output(
                ['dir', '/q', file_path], shell=True, text=True)
            # Extract the line containing the file information
            lines = result.splitlines()
            file_name = os.path.basename(file_path)
            if len(lines) >= 6:
                for line in lines:
                    if file_name in line:
                        exploded_line = line.split(' ')
                        owner_name = exploded_line[-2]
        except subprocess.CalledProcessError:
            owner_name = ""
    else:
        try:
            from pwd import getpwuid
            # Use the 'os.stat()' method to get the file owner on non-Windows systems
            file_stat = os.stat(file_path)
            # You can also use pwd.getpwuid(file_stat.st_uid).pw_name to get the username
            owner_name = file_stat.st_uid
            owner_name = getpwuid(owner_name).pw_name + \
                " (" + str(owner_name) + ")"
        except OSError:
            owner_name = ""

    return owner_name


def RedactData(input_string):
    if len(input_string) < 3:
        return input_string

    # Calculate the number of characters to redact in the middle (half of the length)
    redact_count = len(input_string) // 2

    # Split the input string into two parts: before and after the middle
    middle_start = len(input_string) // 2 - redact_count // 2
    middle_end = len(input_string) // 2 + redact_count // 2

    before_middle = input_string[:middle_start]
    middle = input_string[middle_start:middle_end]
    after_middle = input_string[middle_end:]

    # Redact the middle part
    redacted_middle = "*" * len(middle)

    # Concatenate the parts back together
    redacted_string = before_middle + redacted_middle + after_middle

    return redacted_string


def get_connection(args):
    if args.connection:
        if os.path.exists(args.connection):
            with open(args.connection, 'r') as file:
                connections = yaml.safe_load(file)
                return connections
        else:
            print_error(args, f"Connection file not found: {args.connection}")
            exit(1)
    elif args.connection_json:
        try:
            connections = json.loads(args.connection_json)
            return connections
        except json.JSONDecodeError as e:
            print_error(args, f"Error parsing JSON: {e}")
            exit(1)
    else:
        print_error(
            args, "Please provide a connection file using --connection flag or connection details using --connection-json flag")
        exit(1)


# Global cache for fingerprint data
_fingerprint_cache = None


def get_fingerprint_file(args=None):
    global _fingerprint_cache
    if _fingerprint_cache:
        return _fingerprint_cache

    if args and type(args) == argparse.Namespace and args.fingerprint:
        if os.path.exists(args.fingerprint):
            with open(args.fingerprint, 'r') as file:
                _fingerprint_cache = yaml.safe_load(file)
                # NEW: Validate patterns after loading
                _validate_fingerprint_patterns(args, _fingerprint_cache)
                return _fingerprint_cache
        else:
            if args:
                print_error(
                    args, f"Fingerprint file not found: {args.fingerprint}")
            exit(1)
    elif args and type(args) == dict and 'fingerprint' in args:
        _fingerprint_cache = args['fingerprint']
        return _fingerprint_cache
    else:
        # Check config/fingerprint.yml first (enhanced version with 90+ patterns)
        if os.path.exists('config/fingerprint.yml'):
            try:
                with open('config/fingerprint.yml', 'r') as file:
                    _fingerprint_cache = yaml.safe_load(file)
                    _validate_fingerprint_patterns(args, _fingerprint_cache)
                    return _fingerprint_cache
            except Exception:
                pass

        # Fall back to fingerprint.yml in CWD
        if os.path.exists('fingerprint.yml'):
            try:
                with open('fingerprint.yml', 'r') as file:
                    _fingerprint_cache = yaml.safe_load(file)
                    # NEW: Validate patterns after loading
                    _validate_fingerprint_patterns(args, _fingerprint_cache)
                    return _fingerprint_cache
            except Exception:
                pass

        file_path = "https://github.com/rohitcoder/hawk-eye/raw/main/fingerprint.yml"
        try:
            response = requests.get(file_path, timeout=10)
            if args:
                print_info(
                    args, f"Downloading default fingerprint.yml from {file_path}")
            if response.status_code == 200:
                with open('fingerprint.yml', 'wb') as file:
                    file.write(response.content)
                _fingerprint_cache = yaml.safe_load(response.content)
                # NEW: Validate patterns after loading
                _validate_fingerprint_patterns(args, _fingerprint_cache)
                return _fingerprint_cache
            else:
                if args:
                    print_error(
                        args, "Unable to download default fingerprint.yml please provide your own fingerprint file using --fingerprint flag")
                exit(1)
        except Exception:
            if args:
                print_error(
                    args, "Unable to download default fingerprint.yml please provide your own fingerprint file using --fingerprint flag")
            exit(1)


def _validate_fingerprint_patterns(args, patterns):
    """Validate fingerprint patterns for common issues"""
    if not patterns:
        return

    # Check 1: Warn about case-sensitive patterns that might miss data
    case_sensitive_patterns = []
    for name, regex in patterns.items():
        if not regex.startswith('(?i)') and any(char.isalpha() for char in regex):
            case_sensitive_patterns.append(name)

    if case_sensitive_patterns and args:
        print_info(
            args, f"ℹ️  Note: {len(case_sensitive_patterns)} patterns are case-sensitive (may miss lowercase variants)")
        if hasattr(args, 'debug') and args.debug:
            print_debug(
                args, f"Case-sensitive patterns: {', '.join(case_sensitive_patterns[:10])}")

    # Check 2: Validate regex compilation
    invalid_patterns = []
    for name, regex in patterns.items():
        try:
            re.compile(regex)
        except re.error as e:
            invalid_patterns.append(f"{name}: {e}")
            print_error(args, f"❌ Invalid regex in pattern '{name}': {e}")

    if invalid_patterns:
        print_error(
            args, f"Found {len(invalid_patterns)} invalid regex patterns - scan may fail")

    # Check 3: Pattern statistics
    total_patterns = len(patterns)
    if args:
        print_success(
            args, f"✅ Loaded {total_patterns} patterns from fingerprint.yml")


def print_banner(args):
    line1 = "+ ================================================== +"
    line2 = "+ [bold yellow]H[/bold yellow].[bold yellow]A[/bold yellow].[bold yellow]W[/bold yellow].[bold yellow]K[/bold yellow] [bold yellow]Eye[/bold yellow] - [bold blue]Highly Advanced Watchful Keeper Eye[/bold blue] +"
    line3 = "+ ================================================== +"
    line4 = ""
    line5 = "Hunt for Secrets & PII Data, like never before!"
    line6 = "A Tool by [bold red]Rohit Kumar (@rohitcoder)[/bold red]"

    banner = f"""
                              ⢀⣀⣀⣀⣀⣤⠶⣒⣛⣛⡲⠶⣶⣶⠤⣤⣀⡀
                            ⣠⠴⢻⣭⣶⣶⣶⣶⣶⣦⣽⣿⣿⣿⣷⣾⣥⣝⣶⣭⣗⣲⡕⣠⡀
                          ⢀⣨⣷⣺⣵⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⣵⣝⣦⡀
                        ⣠⡾⡫⢵⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡟⣿⣿⡿⣿⣿⣞⣿⡄
                       ⢼⣟⣯⣾⣿⣿⣿⣿⣿⡍⠉⠙⠛⠟⠿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡌⣿⣷⢸⣿⣿⣾⣿
                      ⢠⢫⣾⣿⣿⣿⣿⣿⠋⠁ ⠠     ⠉⠻⢿⣿⣿⣿⣿⣿⠟⢀⣿⠟⣘⣿⣿⣿⣿
                     ⣰⣏⣿⣿⣿⣿⣿⣿⣿⣇ ⠠⡖⠒⣲⣶⣶⣶⣦⣤⣄⡛⠛⠛⢛⣁⣤⡽⠿⠋⠉⠉ ⠙⠛⠒⠦⣄
                     ⢿⡅⣼⣿⣿⣿⣿⣿⣿⣿⠗ ⠱⡀⢿⣿⣤⣧⡼⢻⢻⣿⠟⠉⢻⠟⠁⡰⠟⠁       ⠈⢳⡄
                    ⢀⡾⣼⣿⣿⣿⣿⣿⣿⣿⣿⣿  ⠈⠓⠭⠿⠭⠄⡊⠤⠊⠎⢢⠏              ⢳
                    ⣼⢱⣿⢿⣿⣿⣿⣿⣿⣿⣿⣿⣄⡀    ⡀⢀    ⡜⢠⢠⢀⢀          ⠈⣾⡄	 {line1}
                    ⣿⣨⢅⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣷⡶⠄⣀⣀⡀  ⡠⢊⣀   ⠈           ⢿⡇	 {line2}
                   ⢀⡴⣽⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⠟⠁ ⡘⠠⠋⢀⠤⢒⣒⣒⣒⠢⠤⢤⣄⣀⣀⣀⡀  ⣼⠁   {line3}
                   ⣎⠾⣻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠆   ⠱⢄⢘⣡⠔⠶⢴⣒⣢⣤⡤⠖⠛⡏⠉⠉⠉⢣⢀⡟    {line4}
                  ⠸⣴⣳⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⠿⠛⠁ ⠐  ⢀⠒⣭⠅    ⠰⠌⠁ ⡀⣷    ⠛     {line5}
                 ⢀⡼⣱⣿⣿⣿⣿⣿⣿⣿⡿⣻⣿⣿⣷⡦⠄    ⠈ ⠈⠁ ⠂    ⠈  ⢀⣾⣧⣿⠁         {line6}
                ⣠⣿⢃⢋⣽⣿⣿⣿⣿⣿⣿⠵⣿⡿⣿⣿⣶⡿⣃⣀⣤⣴⣇⣠⣾⡇       ⢠⡗⣸⣿⣿⣿⠆
              ⢀⣾⣿⢟⣵⣿⣿⣿⣿⣿⣿⣿⣿⣿⣟⣶⣿⣿⣿⣿⣿⢿⣿⣿⢿⣿⡟  ⢀⡄   ⢀⣾⣿⣿⣿⣿⣯
             ⢀⣿⣿⣯⣶⢖⣸⣟⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣃⣿⣭⣾⣿⣯⣤⣴⣾⣿⢇⢌⣾⡆⣼⣿⣿⣿⣿⣏⢃⠃
            ⠠⣫⠾⣻⣿⢿⣿⠫⠾⠿⠿⣿⡛⣫⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡟⣷⣄⡀
            ⠐⠁⣼⠟⢡⣿⠿⣿⣿⣿⡿⢟⣡⢖⣾⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣛⣻⢹⣇⡀
             ⠐⠁⢀⠟⢁⣾⣿⠿⣿⣾⣿⢯⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⡿⡿⣿⢓⣛⣿⣿⡿⢆⡀
               ⠊⢀⡞⠉ ⣴⠟⣻⣷⣶⣶⣾⣭⣽⣿⣭⣵⡾⣻⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣿⣉⠿⣩⣣⣿⣷⡞⢟⠟⣿⢛⠆⠈⠂
                ⠈  ⠐⠁⣼⠟⢻⣿⣿⡿⠿⢿⣿⣿⣟⣟⣻⣿⣻⣿⣭⢇⡿⣛⢻⣿⣋⢿⢣⣿⣿⢿⣿⣿⡇   ⠙⠈⠌
                    ⠈⠁ ⠛⠋ ⣀⣴⣿⣿⣿⡿⢿⣿⣿⣿⣿⣿⣬⣾⣿⣟⣷⣿⣶⣿⠟⠁ ⣿⡟
                        ⠐⠞⠛⠉    ⣼⠿⠋ ⠉⢹⣿⡿⣿⣿⢿⣿⠏⠃   ⠙
                                    ⢀⣾⠏⢠⡿⠁⡾⠋
                                    ⠈⠁ ⠈
"""
    if args.quiet:
        args.shutup = True
    if not args.shutup:
        console.print(banner)



def should_exclude_file(args, file_name, exclude_patterns):
    _, extension = os.path.splitext(file_name)
    if extension in exclude_patterns:
        print_debug(
            args, f"Excluding file: {file_name} because of extension: {extension}")
        return True

    for pattern in exclude_patterns:
        if pattern in file_name:
            print_debug(
                args, f"Excluding file: {file_name} because of pattern: {pattern}")
            return True
    return False


def should_exclude_folder(folder_name, exclude_patterns):
    for pattern in exclude_patterns:
        if pattern in folder_name:
            return True
    return False


def list_all_files_iteratively(args, path, exclude_patterns):
    for root, dirs, files in os.walk(path, topdown=True):
        dirs[:] = [d for d in dirs if not should_exclude_folder(
            os.path.join(root, d), exclude_patterns)]

        for file in files:
            if not should_exclude_file(args, file, exclude_patterns):
                yield os.path.join(root, file)


def scan_file(file_path, args=None, source=None):
    content = ''
    is_archive = False
    # Check if the file is an image
    if file_path.lower().endswith(('.png', '.jpg', '.jpeg', '.gif', '.bmp')):
        content = enhance_and_ocr(file_path)
    # Check if the file is a PDF document
    elif file_path.lower().endswith('.pdf'):
        content = read_pdf(args, file_path)
    # Check if the file is an office document (Word, Excel, PowerPoint)
    elif file_path.lower().endswith(('.docx', '.xlsx', '.pptx')):
        content = read_office_document(args, file_path)
    # Check if the file is an archive (zip, rar, tar, tar.gz)
    elif file_path.lower().endswith(('.mp4', '.avi', '.mov', '.mkv')):
        content = read_video(args, file_path)
    elif file_path.lower().endswith(('.zip', '.rar', '.tar', '.tar.gz')):
        # this is archive, so we need to extract it and find pii from it, and return matched_strings
        matched_strings = find_pii_in_archive(args, file_path, source)
        is_archive = True
    else:
        # For other file types, read content normally
        with open(file_path, 'rb') as file:
            # Attempt to decode using UTF-8, fallback to 'latin-1' if needed
            content = file.read().decode('utf-8', errors='replace')

    if not is_archive:
        matched_strings = match_strings(args, content, source)
    return matched_strings


def read_match_strings(args, file_path, source):
    print_info(args, f"Scanning file: {file_path} for Source: {source}")
    try:
        matched_strings = scan_file(file_path, args, source)
    except Exception as e:
        print_debug(args, f"Error in read_match_strings: {e}")
        matched_strings = []
    return matched_strings


def read_pdf(args, file_path):
    content = ''
    try:
        # Read content from PDF document
        with open(file_path, 'rb') as file:
            pdf_reader = PyPDF2.PdfReader(file)
            # Use len() instead of deprecated numPages
            for page_num in range(len(pdf_reader.pages)):
                page = pdf_reader.pages[page_num]
                try:
                    content += page.extract_text()
                except UnicodeDecodeError:
                    # Handle decoding errors by trying a different encoding
                    content += page.extract_text(encoding='latin-1')
    except Exception as e:
        print_debug(args, f"Error in read_pdf: {e}")
    return content


def process_frame(frame, frame_num, args):
    """
    Process a single frame: convert it to grayscale and run OCR.

    :param frame: The video frame to process.
    :param frame_num: The frame number (for debugging).
    :param args: Arguments for debugging or additional settings.
    :return: Extracted text from the frame.
    """
    try:
        gray_frame = cv2.cvtColor(frame, cv2.COLOR_BGR2GRAY)

        # Resize for faster OCR if resolution is not critical
        small_frame = cv2.resize(
            gray_frame, (gray_frame.shape[1] // 2, gray_frame.shape[0] // 2))

        # Apply thresholding (optional, improves OCR on some images)
        _, thresh_frame = cv2.threshold(
            small_frame, 150, 255, cv2.THRESH_BINARY)

        # Perform OCR with optimized configuration
        custom_config = r'--oem 3'  # PSM 6 for uniform block of text
        text = pytesseract.image_to_string(thresh_frame, config=custom_config)

        if args.debug:
            print(f"Processed frame {frame_num}")

        return text.strip()
    except Exception as e:
        if args.debug:
            print(f"Error processing frame {frame_num}: {e}")
        return ""


def process_frames_parallel(frames, args):
    with ProcessPoolExecutor(max_workers=4) as executor:  # Use multiple processes
        futures = [executor.submit(process_frame, frame, i, args)
                   for i, frame in enumerate(frames)]
        return [future.result() for future in futures]


def read_video(args, file_path, frame_interval=30, max_workers=10):
    """
    Extract text from a video file by applying OCR on its frames.

    :param args: Arguments for debugging or additional settings.
    :param file_path: Path to the video file.
    :param frame_interval: Interval to capture frames (default is every 30 frames).
    :param max_workers: Number of parallel workers (default is 4).
    :return: Extracted text content from video frames.
    """
    content = ''
    try:
        # Open the video file
        cap = cv2.VideoCapture(file_path)

        if not cap.isOpened():
            raise ValueError(f"Cannot open video file: {file_path}")

        frame_count = int(cap.get(cv2.CAP_PROP_FRAME_COUNT))
        frame_rate = cap.get(cv2.CAP_PROP_FPS)
        print_debug(
            args, f"Processing {frame_count} frames at {frame_rate} FPS")

        futures = []
        processed_frames = 0

        # Create a ThreadPoolExecutor for parallel processing
        with ThreadPoolExecutor(max_workers=max_workers) as executor:
            for frame_num in range(frame_count):
                ret, frame = cap.read()
                if not ret:
                    break

                # Process only every `frame_interval`-th frame
                if frame_num % frame_interval == 0:
                    print_debug(
                        args, f"Submitting frame {frame_num}/{frame_count} for processing")
                    # Submit the frame to the thread pool for processing
                    futures.append(executor.submit(
                        process_frame, frame, frame_num, args))
                    processed_frames += 1

            # Wait for all submitted frames to complete and gather the results
            for future in as_completed(futures):
                text = future.result()
                content += text + '\n'

        cap.release()
        print_debug(
            args, f"Processed {processed_frames} frames out of {frame_count}")

    except Exception as e:
        print_debug(args, f"Error in read_video: {e}")

    return content


def read_office_document(args, file_path):
    content = ''
    try:
        # Check the file type and read content accordingly
        if file_path.lower().endswith('.docx'):
            # Read content from Word document
            doc = Document(file_path)
            for paragraph in doc.paragraphs:
                content += paragraph.text + '\n'
        elif file_path.lower().endswith('.xlsx'):
            # Read content from Excel spreadsheet
            workbook = load_workbook(file_path)
            for sheet_name in workbook.sheetnames:
                sheet = workbook[sheet_name]
                for row in sheet.iter_rows():
                    for cell in row:
                        if cell.value is not None:
                            content += str(cell.value) + '\n'
        elif file_path.lower().endswith('.pptx'):
            # Read content from PowerPoint presentation
            # You can add specific logic for PowerPoint if needed
            pass
    except Exception as e:
        print_debug(args, f"Error in read_office_document: {e}")
    return content


def find_pii_in_archive(args, file_path, source):
    content = []
    # Create a temporary directory to extract the contents of the archive
    with tempfile.TemporaryDirectory() as tmp_dir:
        # Extract the contents of the archive based on the file extension
        if file_path.lower().endswith('.zip'):
            patoolib.extract_archive(file_path, outdir=tmp_dir)
        elif file_path.lower().endswith('.rar'):
            patoolib.extract_archive(file_path, outdir=tmp_dir)
        elif file_path.lower().endswith('.tar'):
            with tarfile.open(file_path, 'r') as tar:
                try:
                    tar.extractall(tmp_dir, filter='data')
                except TypeError:
                    tar.extractall(tmp_dir)
        elif file_path.lower().endswith('.tar.gz'):
            with tarfile.open(file_path, 'r:gz') as tar:
                try:
                    tar.extractall(tmp_dir, filter='data')
                except TypeError:
                    tar.extractall(tmp_dir)
        # Iterate over all files in the temporary directory
        for root, dirs, files in os.walk(tmp_dir):
            for file in files:
                file_path = os.path.join(root, file)
                data = read_match_strings(args, file_path, source)
                for d in data:
                    content.append(d)
        # TemporaryDirectory context manager handles cleanup — no manual rmtree needed
    return content


def getFileData(file_path):
    try:
        # Get file metadata
        file_stat = os.stat(file_path)

        # Get the username of the file's creator/owner (Unix only)
        creator_name = ''
        group_name = ''
        try:
            import pwd
            import grp
            try:
                creator_name = pwd.getpwuid(file_stat.st_uid).pw_name
            except KeyError:
                creator_name = str(file_stat.st_uid)
            try:
                group_name = grp.getgrgid(file_stat.st_gid).gr_name
            except KeyError:
                group_name = str(file_stat.st_gid)
        except ImportError:
            pass  # Windows — pwd/grp not available

        # Convert timestamps to human-readable format
        created_time = datetime.datetime.fromtimestamp(
            file_stat.st_ctime).strftime("%Y-%m-%d %H:%M:%S")
        modified_time = datetime.datetime.fromtimestamp(
            file_stat.st_mtime).strftime("%Y-%m-%d %H:%M:%S")
        accessed_time = datetime.datetime.fromtimestamp(
            file_stat.st_atime).strftime("%Y-%m-%d %H:%M:%S")

        # Create a dictionary with the file information
        file_info = {
            "absolute_path": os.path.abspath(file_path),
            "filename": os.path.basename(file_path),
            "extension": os.path.splitext(file_path)[1],
            "size_bytes": file_stat.st_size,
            "permissions_octal": oct(file_stat.st_mode)[-3:],
            "owner": creator_name,
            "group": group_name,
            "created_time": created_time,
            "modified_time": modified_time,
            "accessed_time": accessed_time,
        }
        return file_info

    except FileNotFoundError:
        return {"error": "File not found"}
    except Exception as e:
        return {"error": str(e)}


def SlackNotify(msg, args):
    connections = get_connection(args)
    if not args.no_write:
        db = TinyDB('previous_alerts.json')

    if 'notify' in connections:
        notify_config = connections['notify']
        # Check if suppress_duplicates is set to True
        suppress_duplicates = notify_config.get('suppress_duplicates', False)
        original_msg = msg
        if suppress_duplicates and not args.no_write:
            # Calculate the hash of the message
            # check if "msg" has "Message Link" in any line, then remove that complete line
            if "Message Link" in msg:
                msg = msg.split("\n")
                msg = [line for line in msg if "Message Link" not in line]
                msg = "\n".join(msg)

            msg_hash = calculate_msg_hash(msg)
            # Check if the message hash already exists in the previous alerts database
            alert_query = Query()
            if db.search(alert_query['msg_hash'] == msg_hash):
                print_info(
                    args, "Duplicate message detected. Skipping webhook trigger.")
                return

        slack_config = notify_config.get('slack', {})
        webhook_url = slack_config.get('webhook_url', '')
        if webhook_url and webhook_url.startswith('https://hooks.slack.com/services/'):
            try:
                payload = {
                    'text': original_msg,
                }
                headers = {'Content-Type': 'application/json'}
                requests.post(webhook_url, data=json.dumps(
                    payload), headers=headers)
                if suppress_duplicates and not args.no_write:
                    # Store the message hash in the previous alerts database
                    db.insert({'msg_hash': msg_hash})
            except Exception as e:
                print_error(args, f"An error occurred: {str(e)}")


def evaluate_severity(json_data, rules):
    """Evaluate severity based on PII TYPE, not volume (Phase 3 fix)"""
    pattern_name = json_data.get('pattern_name', '').lower()

    # TYPE-BASED SEVERITY (not volume-based)
    # Sensitive government IDs and financial data
    if any(kw in pattern_name for kw in ['ssn', 'aadhaar', 'aadhar', 'pan', 'pancard', 'credit_card', 'debit_card', 'passport']):
        severity = 'Critical'
        description = 'Sensitive PII detected (government ID or financial data)'
    # Credentials and secrets
    elif any(kw in pattern_name for kw in ['aws_key', 'api_key', 'secret', 'password', 'token', 'private_key', 'access_key']):
        severity = 'Critical'
        description = 'Credentials or secrets detected'
    # Personal contact information
    elif any(kw in pattern_name for kw in ['email', 'phone', 'mobile']):
        severity = 'High'
        description = 'Personal contact information detected'
    # Other potential PII
    else:
        severity = 'Medium'
        description = 'Potentially sensitive data detected'

    json_data['severity'] = severity
    json_data['severity_description'] = description
    return json_data


def enhance_and_ocr(image_path):
    # Load the image
    original_image = Image.open(image_path)

    # Enhance the image (you can adjust enhancement factors as needed)
    enhanced_image = enhance_image(original_image)

    # Save the enhanced image to a unique temp file to avoid concurrent scan conflicts
    tmp_img = tempfile.NamedTemporaryFile(suffix='.png', delete=False)
    tmp_img.close()
    enhanced_image.save(tmp_img.name)

    # Perform OCR on the enhanced image
    ocr_text = perform_ocr(enhanced_image)
    # delete the enhanced image
    os.remove(tmp_img.name)

    return ocr_text


def enhance_image(image):
    # Convert to grayscale
    grayscale_image = image.convert('L')

    # Increase contrast
    contrast_enhancer = ImageEnhance.Contrast(grayscale_image)
    contrast_factor = 2.0  # Adjust as needed
    contrast_enhanced_image = contrast_enhancer.enhance(contrast_factor)

    # Apply thresholding
    threshold_value = 100  # Adjust as needed
    thresholded_image = contrast_enhanced_image.point(
        lambda x: 0 if x < threshold_value else 255)

    # Reduce noise (optional)
    denoised_image = cv2.fastNlMeansDenoising(np.array(
        thresholded_image), None, h=10, templateWindowSize=7, searchWindowSize=21)

    return Image.fromarray(denoised_image)


def perform_ocr(image):
    # Use Tesseract OCR
    ocr_text = pytesseract.image_to_string(image)

    return ocr_text


def get_jira_accId(args, email):
    config = get_connection(args)
    jira_config = config.get('notify', {}).get('jira', {})
    server_url = jira_config.get('server_url')
    username = jira_config.get('username')
    api_token = jira_config.get('api_token')
    db = TinyDB('user_ids.json')
    user_query = Query()

    # Check if the accountId is already cached
    cached_user = db.search(user_query.email == email)
    if cached_user:
        print_debug(
            args, f"Using cached accountId for {email}: {cached_user[0]['accountId']}")
        return cached_user[0]['accountId']

    # Fetch accountId from Jira
    url = f"{server_url}/rest/api/2/user/search?query={email}"
    auth = (username, api_token)
    headers = {"Content-Type": "application/json"}

    response = requests.get(url, auth=auth, headers=headers)
    if response.status_code == 200:
        data = response.json()
        if len(data):
            account_id = data[0]['accountId']
            print_debug(args, f"Found accountId for {email}: {account_id}")

            # Cache the accountId
            db.insert({"email": email, "accountId": account_id})
            return account_id
        else:
            print_debug(args, f"No accountId found for {email}.")
            return None
    else:
        print_debug(
            args, f"Failed to fetch accountId for {email}: {response.status_code}, {response.text}")
        return None


def create_jira_ticket(args, issue_data, message):
    orig_msg = message
    config = get_connection(args)
    if not args.no_write:
        db = TinyDB('previous_alerts.json')
        if 'notify' in config:
            notify_config = config['notify']
            # Check if suppress_duplicates is set to True
            suppress_duplicates = notify_config.get(
                'suppress_duplicates', False)
            if suppress_duplicates and not args.no_write:
                # Calculate the hash of the message
                # check if "msg" has "Message Link" in any line, then remove that complete line
                if "Message Link" in message:
                    message = message.split("\n")
                    message = [
                        line for line in message if "Message Link" not in line]
                    message = "\n".join(message)

                msg_hash = calculate_msg_hash(message)
                # Check if the message hash already exists in the previous alerts database
                alert_query = Query()
                if db.search(alert_query['msg_hash'] == msg_hash):
                    print_info(
                        args, "Duplicate message detected. Skipping ticket creation")
                    return

    """Creates a Jira ticket using the provided configuration and issue data."""
    jira_config = config.get('notify', {}).get('jira', {})

    # Check if Jira is enabled
    if not jira_config.get('username') or jira_config.get('username') == '':
        print_debug(
            args, "Jira ticket creation is disabled in the configuration.")
        return

    # Extract Jira config details
    server_url = jira_config.get('server_url')
    evaluated_result = evaluate_severity(issue_data, config)
    severity = evaluated_result.get('severity')
    severity_description = evaluated_result.get('severity_description')
    username = jira_config.get('username')
    api_token = jira_config.get('api_token')
    project = jira_config.get('project')
    default_issue_type = jira_config.get('issue_type')
    issue_fields = jira_config.get('issue_fields', {})
    total_matches = len(issue_data.get('matches', []))
    summary = "Found " + str(total_matches) + " " + issue_data.get(
        'pattern_name') + " in " + issue_data.get('data_source')
    description_template = issue_fields.get('description_template', '')
    orig_msg = orig_msg + "\n\n" + "Severity: " + severity + \
        "\n" + "Severity Description: " + severity_description
    description = description_template.format(details=orig_msg, **issue_data)
    print("severity - ", severity)
    payload = {
        "fields": {
            "project": {"key": project},
            "summary": summary,
            "description": description,
            "issuetype": {"name": default_issue_type},
            "priority": {"name": severity},
        }
    }

    # Check if the assignee is specified in the configuration
    assignee = jira_config.get('assignee')
    if assignee:
        payload['fields']['assignee'] = {
            "accountId": get_jira_accId(args, assignee)}

    labels = jira_config.get('labels')
    if labels:
        payload['fields']['labels'] = labels

    # Send request to Jira API
    url = f"{server_url}/rest/api/latest/issue"
    auth = (username, api_token)
    headers = {"Content-Type": "application/json"}

    response = requests.post(url, json=payload, auth=auth, headers=headers)
    if response.status_code == 201:
        print_debug(
            args, f"Jira ticket created successfully: {response.json().get('key')}")
    else:
        print_debug(
            args, f"Failed to create Jira ticket: {response.status_code} - {response.text}")

def match_strings(args, content, source='text', pii_types=None):
    if args and hasattr(args, 'connection'):
        connections = get_connection(args)
        if 'notify' in connections:
            redacted = connections.get('notify', {}).get('redacted', False)
        # Auto-read pii_types from connection config when not explicitly passed.
        # This gives ALL connectors selective PII filtering automatically.
        if pii_types is None:
            conn_pii_types = connections.get('pii_types', [])
            if conn_pii_types:
                pii_types = conn_pii_types

    patterns = get_fingerprint_file(args)

    # Filter fingerprint patterns to only user-selected PII types
    if pii_types and patterns:
        from sdk.pii_type_mapping import filter_fingerprint_patterns
        patterns = filter_fingerprint_patterns(patterns, pii_types)

    # Resolve Presidio entity list (None = all, list = selective)
    presidio_entities = None
    if pii_types:
        from sdk.pii_type_mapping import get_presidio_entities_for_pii_types
        presidio_entities = get_presidio_entities_for_pii_types(pii_types)

    all_findings = []
    seen = set()

    # Run ContextAwareScanner with fingerprint patterns (regex + entropy + context)
    if patterns:
        try:
            from hawk_scanner.internals.scanner_engine import ContextAwareScanner
            ctx_scanner = ContextAwareScanner()
            regex_results = ctx_scanner.scan(content, patterns, source)
            for f in regex_results:
                raw = f['matches'][0] if f.get('matches') else ''
                key = (f['pattern_name'], raw)
                if key not in seen:
                    seen.add(key)
                    all_findings.append({
                        'pattern_name': f['pattern_name'],
                        'confidence_score': f.get('confidence_score', 50) / 100.0,
                        'matches': f.get('matches', []),
                        'sample_text': f.get('sample_text', ''),
                    })
        except Exception as e:
            if args:
                print_debug(args, f"ContextAwareScanner error: {e}")

    # Run Presidio for AI-powered detection
    try:
        from sdk.engine import SharedAnalyzerEngine
        wrapper = SharedAnalyzerEngine()
        engine = wrapper.get_engine()
        presidio_results = engine.analyze(
            text=content, entities=presidio_entities, language="en"
        )

        for r in presidio_results:
            raw = content[r.start:r.end].strip()
            if not raw:
                continue
            pattern_name = r.entity_type
            if pattern_name == "IN_PAN":
                pattern_name = "PAN"
            key = (pattern_name, raw)
            if key not in seen:
                seen.add(key)
                all_findings.append({
                    'pattern_name': pattern_name,
                    'confidence_score': r.score,
                    'matches': [raw],
                    'sample_text': raw,
                    'start': r.start,
                    'end': r.end,
                })
    except Exception as e:
        if args:
            print_debug(args, f"Presidio analysis error: {e}")

    for finding in all_findings:
        if args:
            print_debug(
                args,
                f"Found pattern: {finding['pattern_name']} Score: {finding['confidence_score']}"
            )

    return all_findings
