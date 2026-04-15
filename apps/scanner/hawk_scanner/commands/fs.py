from hawk_scanner.internals import system
from hawk_scanner.internals.binary_detection import is_text_file, get_binary_file_stats
from hawk_scanner.internals.validation_integration import validate_findings
import os
import concurrent.futures
import time


def process_file(args, file_path, key, results):
    if not is_text_file(file_path):
        if hasattr(args, 'verbose') and args.verbose:
            system.print_info(args, f"Skipped binary file: {file_path}")
        return

    from hawk_scanner.internals.match_fix import match_strings

    with open(file_path, "r", errors="ignore") as f:
        content = f.read()

    matches = match_strings(args, content, source='fs')
    file_data = system.getFileData(file_path)
    if matches:
        validated_matches = validate_findings(matches, args)
        if validated_matches:
            for match in validated_matches:
                results.append({
                    'host': 'This PC',
                    'file_path': file_path,
                    'pattern_name': match['pattern_name'],
                    'matches': match['matches'],
                    'sample_text': match.get('sample_text', ''),
                    'profile': key,
                    'data_source': 'fs',
                    'file_data': file_data,
                    'validation_method': match.get('validation_method', {}),
                    'validated': True
                })


def execute(args):
    results = []
    connections = system.get_connection(args)
    if 'sources' in connections:
        sources_config = connections['sources']
        fs_config = sources_config.get('fs')
        if fs_config:
            for key, config in fs_config.items():
                if 'path' not in config:
                    system.print_error(
                        args, f"Path not found in fs profile '{key}'")
                    continue
                path = config.get('path')
                if not os.path.exists(path):
                    system.print_error(args, f"Path '{path}' does not exist")

                exclude_patterns = fs_config.get(
                    key, {}).get('exclude_patterns', [])
                start_time = time.time()
                # CHECK If file or directory
                if os.path.isfile(path):
                    files = [path]
                else:
                    # Convert generator to list to allow len() operations
                    files = list(system.list_all_files_iteratively(
                        args, path, exclude_patterns))

                # NEW: Analyze binary vs text files
                file_stats = get_binary_file_stats(files)
                system.print_info(
                    args, f"File analysis: {file_stats['text']} text files, {file_stats['binary']} binary files skipped")

                # Use ThreadPoolExecutor for parallel processing
                file_count = 0
                with concurrent.futures.ThreadPoolExecutor() as executor:
                    futures = []
                    for file_path in files:
                        file_count += 1
                        futures.append(executor.submit(
                            process_file, args, file_path, key, results))

                    # Wait for all tasks to complete
                    for future in concurrent.futures.as_completed(futures):
                        try:
                            future.result()
                        except Exception as e:
                            print("THREAD ERROR:", e)
                            import traceback
                            traceback.print_exc()
                end_time = time.time()
                system.print_info(
                    args, f"Time taken to analyze {file_stats['text']} text files: {end_time - start_time} seconds")
        else:
            system.print_error(
                args, "No filesystem 'fs' connection details found in connection.yml")
    else:
        system.print_error(
            args, "No 'sources' section found in connection.yml")
    return results
