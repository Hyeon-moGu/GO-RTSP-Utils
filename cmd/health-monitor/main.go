package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"go-rtsp-tools/internal/config"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/pion/rtp"
)

type CheckResult struct {
	Name    string
	URL     string
	Success bool
	Message string
}

func main() {
	config.LoadConfig()

	streams := config.Cfg.RTSPStreams
	parallel := config.Cfg.Health.Parallel
	duration := time.Duration(config.Cfg.Health.ChkSec) * time.Second

	sem := make(chan struct{}, parallel)
	var wg sync.WaitGroup

	for _, stream := range streams {
		sem <- struct{}{}
		wg.Add(1)
		go func(s config.RTSPStream) {
			defer wg.Done()
			defer func() { <-sem }()

			result := CheckStream(s.Name, s.URL, duration)
			log.Println(result)
		}(stream)
	}

	wg.Wait()
	fmt.Println("\n========== 모든 스트림 점검 완료 ==========")
}

func CheckStream(name, url string, duration time.Duration) CheckResult {
	client := &gortsplib.Client{}
	defer client.Close()

	parsedURL, err := base.ParseURL(url)
	if err != nil {
		return CheckResult{name, url, false, "URL 파싱 실패"}
	}

	err = client.Start(parsedURL.Scheme, parsedURL.Host)
	if err != nil {
		return CheckResult{name, url, false, "연결 실패"}
	}

	session, _, err := client.Describe(parsedURL)
	if err != nil {
		return CheckResult{name, url, false, "DESCRIBE 실패"}
	}

	err = client.SetupAll(session.BaseURL, session.Medias)
	if err != nil {
		return CheckResult{name, url, false, "SETUP 실패"}
	}

	var received int
	for _, media := range session.Medias {
		for _, format := range media.Formats {
			client.OnPacketRTP(media, format, func(pkt *rtp.Packet) {
				received++
			})
		}
	}

	_, err = client.Play(nil)
	if err != nil {
		return CheckResult{name, url, false, "PLAY 실패"}
	}

	time.Sleep(duration)

	if received == 0 {
		return CheckResult{name, url, false, "RTP 수신 없음"}
	}

	return CheckResult{name, url, true, fmt.Sprintf("정상 수신 (%d packets)", received)}
}
