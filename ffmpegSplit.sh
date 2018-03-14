#!/bin/sh

VIDEO_NAME=it
FRAMES_SECOND=24

rm data/frames/*.png
rm data/faces/*.png

ffmpeg -i data/videos/${VIDEO_NAME}.mp4 -r ${FRAMES_SECOND}/1 data/frames/%05d_${VIDEO_NAME}.png