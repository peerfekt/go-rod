package rod_test

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"

	"github.com/TommyLeng/go-rod"
	"github.com/TommyLeng/go-rod/lib/proto"
)

//GODEBUG="tracebackancestors=1000" go test -timeout 30s -run ^TestPageScreenCastAvi$ github.com/TommyLeng/go-rod -v -count=1 -parallel=1
func TestPageScreenCastAvi(t *testing.T) {
	g := setup(t)

	page := g.page.MustNavigate(g.srcFile("./fixtures/timer.html"))

	time.Sleep(5 * time.Second)

	videoFrames := []rod.VideoFrame{}
	fps := 100
	time.Sleep(5 * time.Second)

	// ScreenCastRecord listen PageScreenCastFrame and save data into videoFrames
	aviWriter, err := page.ScreenCastRecordAvi("output.avi", &videoFrames, fps) // Only support .avi video file & frame per second
	if err != nil {
		g.Fatal(err)
	}

	// ScreenCastStart start listening ScreenCastRecord
	err = page.ScreenCastStart(50) // Image quality
	if err != nil {
		g.Fatal(err)
	}

	fmt.Println("sleep 10 seconds start: ", time.Now())
	time.Sleep(10 * time.Second)

	err = page.Navigate(g.srcFile("./fixtures/blank.html"))
	if err != nil {
		g.Fatal(err)
	}
	page.MustWaitNavigation()
	page.MustWaitLoad()
	time.Sleep(4 * time.Second)

	// ScreenCastStop stop listening ScreenCastRecord and convert the videoFrames data into avi file
	err = page.ScreenCastStopAvi(aviWriter, &videoFrames, fps)
	if err != nil {
		g.Fatal(err)
	}

	page.MustClose()
}

//Direct put data from screen cast event to ffmpeg stdin
//GODEBUG="tracebackancestors=1000" go test -timeout 60s -run ^TestPageScreenCastDirectMp4$ github.com/TommyLeng/go-rod -v -count=1 -parallel=1
func TestPageScreenCastDirectMp4(t *testing.T) {
	g := setup(t)

	page := g.page.MustNavigate(g.srcFile("./fixtures/timer.html"))

	time.Sleep(5 * time.Second)
	fps := 50

	dataCh := make(chan []byte, 12)

	//cat $(find . -maxdepth 1 -name '*.png' -print | sort | tail -10) | ffmpeg -framerate 25 -i - -vf format=yuv420p -movflags +faststart output.mp4

	cmd := exec.Command("ffmpeg",
		"-y", // Yes to all
		"-f", "image2pipe",
		"-r", strconv.Itoa(fps),
		"-i", "pipe:0", // take stdin as input
		"-an",
		"-vf", "format=yuv420p",
		"-vsync", "1",
		"-movflags", "+faststart",
		"output_direct_pipe.mp4", // output
	)

	cmd.Stderr = os.Stderr // bind log stream to stderr
	stdin, err := cmd.StdinPipe()
	if err != nil {
		g.Fatal(err)
	}

	err = cmd.Start() // Start a process on another goroutine
	if err != nil {
		g.Fatal(err)
	}

	go page.EachEvent(func(e *proto.PageScreencastFrame) {
		err := proto.PageScreencastFrameAck{
			SessionID: e.SessionID,
		}.Call(page)
		if err != nil {
			g.Fatalf("ScreencastFrameAck err:", err)
		}
		dataCh <- e.Data
	})()

	timer := time.NewTicker(time.Second / time.Duration(fps))
	go func() {
		var data []byte
		for {
			select {
			case d, ok := <-dataCh:
				if !ok {
					return
				}
				data = d
			case <-timer.C:
				if len(data) > 0 {
					//pw.Write(data)
					_, err = stdin.Write(data)
					if err != nil {
						//may have write err due to stdin closed
						return
					}
				}
			}
		}
	}()

	everyNthFrame := 1
	qty := 100
	err = proto.PageStartScreencast{
		Format:        proto.PageStartScreencastFormatJpeg,
		Quality:       &qty,
		EveryNthFrame: &everyNthFrame,
	}.Call(page)
	if err != nil {
		g.Fatal(err)
	}

	time.Sleep(15 * time.Second)

	err = page.Navigate(g.srcFile("./fixtures/blank.html"))
	if err != nil {
		g.Fatal(err)
	}
	page.MustWaitNavigation()
	page.MustWaitLoad()
	time.Sleep(4 * time.Second)

	err = proto.PageStopScreencast{}.Call(page)
	if err != nil {
		g.Fatal(err)
	}

	timer.Stop()
	time.Sleep(2 * time.Second)
	close(dataCh)

	err = stdin.Close() // close the stdin, or ffmpeg will wait forever
	if err != nil {
		g.Fatal(err)
	}

	err = cmd.Wait() // wait until ffmpeg finish
	if err != nil {
		g.Log(err)
	}

	page.MustClose()
}

//GODEBUG="tracebackancestors=1000" go test -timeout 60s -run ^TestPageScreenCastMp4$ github.com/TommyLeng/go-rod -v -count=1
func TestPageScreenCastMp4(t *testing.T) {
	g := setup(t)

	page := g.page.MustNavigate(g.srcFile("./fixtures/timer.html"))

	time.Sleep(5 * time.Second)
	videoFrames := []rod.VideoFrame{}
	fps := 50

	// ScreenCastRecord listen PageScreenCastFrame and save data into videoFrames
	err := page.ScreenCastRecordMp4(&videoFrames)
	if err != nil {
		g.Fatal(err)
	}

	// ScreenCastStart start listening ScreenCastRecord
	err = page.ScreenCastStart(100) // Image quality & frame per second
	if err != nil {
		g.Fatal(err)
	}

	fmt.Println("sleep 15 seconds")
	time.Sleep(15 * time.Second)

	err = page.Navigate(g.srcFile("./fixtures/blank.html"))
	if err != nil {
		g.Fatal(err)
	}
	page.MustWaitNavigation()
	page.MustWaitLoad()
	time.Sleep(4 * time.Second)

	// ScreenCastStop stop listening ScreenCastRecord and convert the videoFrames data into mp4 file
	err = page.ScreenCastStopMp4(&videoFrames, "output_use_buffer.mp4", fps)
	if err != nil {
		g.Fatal(err)
	}

	page.MustClose()
}
