#!/bin/bash

usage() {
    echo "Usage: $0 --event '<json>'"
    echo "Example: $0 --event '{\"localFilename\": \"file.mp4\"}'"
    exit 1
}


EVENT=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --event)
        EVENT="$2"
        shift
        shift
        ;;
        *)
        usage
        ;;
    esac
done

if [ -z "$EVENT" ]; then
    usage
fi

name=$(echo "$EVENT" | jq -r '.localFilename')

if [ -z "$name" ]; then
    echo "Error: 'localFilename' key not found or empty in the provided JSON."
    exit 1
fi

if [ -z "$SECRET_MESSAGE" ]; then
    echo "Error: SECRET_MESSAGE environment variable is not set."
    exit 1
fi

extension="${name##*.}"
base="${name%.*}"
tmp_file="${base}_tmp.${extension}"

if [ "$extension" = "flac" ]; then
    ffmpeg -i "$name" -metadata secret="$SECRET_MESSAGE" -c copy -y "$tmp_file";
    mv "$tmp_file" "$name"
    echo "Metadata added and file updated successfully: $name"
else
    echo "Error: Failed to process the file with ffmpeg."
    rm -f "$tmp_file"
    exit 1
fi
