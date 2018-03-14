#!/bin/sh

VIDEO_NAME=it
FRAMES_SECOND=24

OUTPUT=${VIDEO_NAME}_join.mp4

rm data/videos/${OUTPUT}

ffmpeg -framerate ${FRAMES_SECOND} -i data/frames/%05d_${VIDEO_NAME}.png data/videos/${OUTPUT}