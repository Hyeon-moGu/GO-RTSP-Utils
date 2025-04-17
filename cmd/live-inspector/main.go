package main

import (
	"bytes"
	"encoding/xml"
	"flag"
	"fmt"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/bluenviron/gortsplib/v4/pkg/description"
	"github.com/pion/rtp"
)

type MetaEvent struct {
	XMLName xml.Name
	Content string `xml:",innerxml"`
}

func main() {
	flag.Parse()
	if len(flag.Args()) < 1 {
		log.Fatal("사용법: go run main.go <RTSP_URL>")
	}
	rtspURL := flag.Arg(0)

	parsedURL, err := base.ParseURL(rtspURL)
	if err != nil {
		log.Fatalf("URL 파싱 실패: %v", err)
	}

	c := &gortsplib.Client{}
	defer c.Close()

	err = c.Start(parsedURL.Scheme, parsedURL.Host)
	if err != nil {
		log.Fatalf("RTSP 연결 실패: %v", err)
	}

	session, _, err := c.Describe(parsedURL)
	if err != nil {
		log.Fatalf("DESCRIBE 실패: %v", err)
	}

	fmt.Println("============= SDP 정보 =============")
	for i, media := range session.Medias {
		fmt.Printf("Track #%d: %s\n", i+1, media.Type)
		fmt.Printf("  Control: %s\n", media.Control)

		for _, forma := range media.Formats {
			fmt.Printf("  Payload: %d\n", forma.PayloadType())
			if rtpmap := forma.RTPMap(); rtpmap != "" {
				fmt.Printf("  Codec: %s\n", rtpmap)
			}
			if fmtp := forma.FMTP(); len(fmtp) > 0 {
				fmt.Println("  FMTP:")
				for k, v := range fmtp {
					fmt.Printf("    %s = %s\n", k, v)
				}
			}
		}
	}

	err = c.SetupAll(session.BaseURL, session.Medias)
	if err != nil {
		log.Fatalf("SETUP 실패: %v", err)
	}

	fmt.Println("\n============= RTP 수신 시작 =============")

	var mu sync.Mutex
	var pktCount int
	var lastSummary time.Time
	var nalStats = make(map[uint8]int)
	var lastSeq *uint16

	for _, media := range session.Medias {
		formats := media.Formats
		for _, forma := range formats {
			f := forma
			m := media

			c.OnPacketRTP(m, f, func(pkt *rtp.Packet) {
				mu.Lock()
				defer mu.Unlock()

				pktCount++
				now := time.Now()

				// 비디오 트랙 처리
				if m.Type == description.MediaTypeVideo {
					if len(pkt.Payload) > 0 {
						nalType := pkt.Payload[0] & 0x1F
						nalStats[nalType]++
						// fmt.Printf("[VIDEO] Seq=%d TS=%d NAL=%d", pkt.Header.SequenceNumber, pkt.Timestamp, nalType)
					}
				}

				if now.Sub(lastSummary) >= 20*time.Second {
					fmt.Println("\n============= 최근 NAL 통계 (20초) =============")
					for k, v := range nalStats {
						fmt.Printf("NAL %d: %d개\n", k, v)
					}
					nalStats = make(map[uint8]int)
					lastSummary = now
				}

				if m.Type == description.MediaTypeApplication {
					if xmlStart := bytes.Index(pkt.Payload, []byte("<")); xmlStart != -1 {
						raw := pkt.Payload[xmlStart:]
						fmt.Println("\n[META] 수신 메타데이터:")
						lines := strings.Split(string(raw), ">")
						for _, line := range lines {
							if len(line) > 0 {
								fmt.Println(strings.TrimSpace(line + ">"))
							}
						}
					}
				}

				if lastSeq != nil {
					seqGap := pkt.Header.SequenceNumber - *lastSeq
					if seqGap > 1 {
						fmt.Printf("[LOSS] 시퀀스 누락 감지- 예상: %d, 현재: %d (손실 %d)\n",
							*lastSeq+1, pkt.Header.SequenceNumber, seqGap-1)
					}
				}
				val := pkt.Header.SequenceNumber
				lastSeq = &val
			})
		}
	}

	_, err = c.Play(nil)
	if err != nil {
		log.Fatalf("PLAY 실패: %v", err)
	}

	for {
		time.Sleep(1 * time.Second)
	}
}
