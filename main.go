package main

import (
	"bytes"
	"fmt"
	"image"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/machinebox/sdk-go/facebox"
	"github.com/machinebox/sdk-go/videobox"
	"github.com/oliamb/cutter"
)

const faceboxUrl = "http://192.168.99.100:8081/"
const videoboxUrl = "http://192.168.99.100:8080/"

const framesPath = "./data/frames"
const videosPath = "./data/videos"
const facesPath = "./data/faces"

func handler(w http.ResponseWriter, r *http.Request) {

}

func main() {

	processVideo("it.mp4")

	// http.HandleFunc("/", handler)
	// http.ListenAndServe(":8080", nil)

}

func processVideo(videoName string) {

	ffmpegSplitDone := make(chan bool)
	videoIDChan := make(chan string)

	go func(done chan<- bool) {
		log.Println("FFMPEG splitting video into frames")
		runFfmpeg()
		done <- true
	}(ffmpegSplitDone)

	go func(videoIdChan chan<- string) {
		log.Println("videobox request!")
		videoID, err := runVideoBox(videoName)
		if err != nil {
			log.Println(err)
		}
		videoIdChan <- videoID
	}(videoIDChan)

	<-ffmpegSplitDone
	log.Println("FFMPEG done!")

	videoID := <-videoIDChan
	log.Println("videobox done!", videoID)

	// DEV
	// videoID := "5a62199c151fce782b85a6350e56cacc"

	faceboxResult, err := videoFaceboxResults(videoID)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Faces found: ", len(faceboxResult.Faces))

	if len(faceboxResult.Faces) > 0 {

		processFrames(faceboxResult.Faces)

	}

}

func videoFaceboxResults(videoID string) (*videobox.Facebox, error) {

	videoboxClient := videobox.New(videoboxUrl)

	analysis, err := videoboxClient.Results(videoID)

	if err != nil {
		return nil, err
	}

	return analysis.Facebox, nil
}

func runVideoBox(videoName string) (string, error) {

	videoboxClient := videobox.New(videoboxUrl)

	imgBytes, err := ioutil.ReadFile(fmt.Sprintf("%s/%s", videosPath, videoName))
	if err != nil {
		return "", err
	}

	opts := videobox.NewCheckOptions()
	opts.FaceboxThreshold(0.25)

	video, err := videoboxClient.Check(bytes.NewReader(imgBytes), opts)
	if err != nil {
		return "", err
	}

	// Check if finished
	for {
		status, err := videoboxClient.Status(video.ID)
		if err != nil {
			return "", err
		}
		if status.Status == videobox.StatusProcessing {
			log.Println("Processing video ...")
		} else if status.Status == videobox.StatusComplete {
			return video.ID, err
		} else if status.Status == videobox.StatusFailed {
			return video.ID, fmt.Errorf("Error processing video: %v", status.Error)
		}

		time.Sleep(5 * time.Second)
	}
}

func processFrames(faces []videobox.Item) {

	faceboxClient := facebox.New(faceboxUrl)
	log.Println("Ready to process ... ")
	for _, face := range faces {
		log.Println("-> ", face.Key)
		for instanceNum, instance := range face.Instances {
			log.Println("  -> Processing instance:", instanceNum)
			for index := instance.Start; index <= instance.End; index++ {

				filename := fmt.Sprintf("%05d_it.png", index)
				filenamePath := fmt.Sprintf("%s/%s", framesPath, filename)
				log.Println("    -> Reading file: ", filenamePath)
				imgBytes, err := ioutil.ReadFile(filenamePath)
				if err != nil {
					log.Println(err)
					continue
				}

				facesCoords, err := faceboxClient.Check(bytes.NewReader(imgBytes))
				if err != nil {
					log.Println(err)
					continue
				}
				if len(facesCoords) == 0 {
					log.Printf("Faces not detected: %s", filename)
					continue
				}

				log.Printf("Cropping faces, %d found!", len(facesCoords))
				for _, faci := range facesCoords {
					if err := cropFace(filename, faci, bytes.NewReader(imgBytes)); err != nil {
						log.Println(err)
					}
				}
			}
		}
	}
}

func cropFace(frameFileName string, face facebox.Face, imageReader io.Reader) error {

	img, _, err := image.Decode(imageReader)
	if err != nil {
		return fmt.Errorf("Decoding image: ", err)
	}

	croppedImg, err := cutter.Crop(img, cutter.Config{
		Width:  face.Rect.Width,
		Height: face.Rect.Height,
		Anchor: image.Point{face.Rect.Left, face.Rect.Top},
	})
	if err != nil {
		return fmt.Errorf("Cropping image: ", err)
	}

	buf := new(bytes.Buffer)
	if err := png.Encode(buf, croppedImg); err != nil {
		return fmt.Errorf("Encoding image: ", err)
	}

	faceName := fmt.Sprintf("%s/%s_%v_%v.png", facesPath, frameFileName, face.Rect.Top, face.Rect.Left)
	if err := ioutil.WriteFile(faceName, buf.Bytes(), 0644); err != nil {
		return fmt.Errorf("Writing image: ", err)
	}

	log.Println("      -> Face cropped and saved:", faceName)
	return nil
}

func runFfmpeg() {
	cmd := exec.Command("/bin/sh", "ffmpegSplit.sh")
	if err := cmd.Run(); err != nil {
		log.Fatal(err)
	}
}
