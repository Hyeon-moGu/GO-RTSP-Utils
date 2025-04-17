package main

import (
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	"go-rtsp-tools/internal/config"

	"github.com/bluenviron/gortsplib/v4"
	"github.com/bluenviron/gortsplib/v4/pkg/base"
	"github.com/pion/rtp"
)

type StreamStat struct {
	Name         string
	URL          string
	TotalPackets int
	LossCount    int
	LossRate     float64
	IDRCount     int
	HasSPS       bool
	HasPPS       bool
	BitrateKbps  float64
	NALSummary   string
}

func main() {
	config.LoadConfig()
	streams := config.Cfg.RTSPStreams
	parallel := config.Cfg.Monitor.Parallel
	duration := time.Duration(config.Cfg.Monitor.DurationSec) * time.Second

	sem := make(chan struct{}, parallel)
	var wg sync.WaitGroup

	for _, stream := range streams {
		sem <- struct{}{}
		wg.Add(1)
		go func(s config.RTSPStream) {
			defer wg.Done()
			defer func() { <-sem }()

			stat := analyzeStream(s.Name, s.URL, duration)
			log.Printf("[%s] Packets- %d / Loss- %d (%.2f%%) / Bitrate- %.1f kbps / KeyFrames- %d / NAL- %s",
				s.Name, stat.TotalPackets, stat.LossCount, stat.LossRate, stat.BitrateKbps, stat.IDRCount, stat.NALSummary)
		}(stream)
	}

	wg.Wait()
	fmt.Println("\n========== 모든 스트림 분석 완료 ==========")
}

func analyzeStream(name, rtspURL string, duration time.Duration) StreamStat {
	client := &gortsplib.Client{ReadTimeout: 10 * time.Second}
	defer client.Close()

	parsedURL, err := base.ParseURL(rtspURL)
	if err != nil {
		log.Printf("[%s] URL 파싱 실패: %v", name, err)
		return StreamStat{Name: name, URL: rtspURL}
	}

	err = client.Start(parsedURL.Scheme, parsedURL.Host)
	if err != nil {
		log.Printf("[%s] 연결 실패: %v", name, err)
		return StreamStat{Name: name, URL: rtspURL}
	}

	session, _, err := client.Describe(parsedURL)
	if err != nil {
		log.Printf("[%s] DESCRIBE 실패: %v", name, err)
		return StreamStat{Name: name, URL: rtspURL}
	}

	err = client.SetupAll(session.BaseURL, session.Medias)
	if err != nil {
		log.Printf("[%s] SETUP 실패: %v", name, err)
		return StreamStat{Name: name, URL: rtspURL}
	}

	var totalPackets int
	var totalBytes int
	var lossCount int
	var lastSeq *uint16
	nalStats := make(map[uint8]int)
	idrCount := 0
	firstSeqSeen := false

	for _, media := range session.Medias {
		for _, format := range media.Formats {
			client.OnPacketRTP(media, format, func(pkt *rtp.Packet) {
				totalPackets++
				totalBytes += len(pkt.Payload)

				if firstSeqSeen && lastSeq != nil {
					gap := (uint32(pkt.SequenceNumber) - uint32(*lastSeq)) & 0xFFFF
					if gap > 1 && gap < 1000 {
						lossCount += int(gap - 1)
					}
				} else {
					firstSeqSeen = true
				}

				val := pkt.SequenceNumber
				lastSeq = &val

				if len(pkt.Payload) > 0 {
					nalType := pkt.Payload[0] & 0x1F
					nalStats[nalType]++
					if nalType == 5 {
						idrCount++
					}
				}
			})
		}
	}

	_, err = client.Play(nil)
	if err != nil {
		log.Printf("[%s] PLAY 실패: %v", name, err)
		return StreamStat{Name: name, URL: rtspURL}
	}

	start := time.Now()
	time.Sleep(duration)
	elapsed := time.Since(start).Seconds()

	lossRate := 0.0
	if totalPackets+lossCount > 0 {
		lossRate = float64(lossCount) / float64(totalPackets+lossCount) * 100
	}

	bitrateKbps := float64(totalBytes*8) / elapsed / 1000
	summary := summarizeNALTypes(nalStats)

	return StreamStat{
		Name:         name,
		URL:          rtspURL,
		TotalPackets: totalPackets,
		LossCount:    lossCount,
		LossRate:     lossRate,
		IDRCount:     idrCount,
		BitrateKbps:  bitrateKbps,
		NALSummary:   summary,
	}
}

func summarizeNALTypes(stats map[uint8]int) string {
	if len(stats) == 0 {
		return "(no NALs)"
	}
	type nalInfo struct {
		Type uint8
		Desc string
		Cnt  int
	}
	descs := map[uint8]string{
		1:  "P",
		5:  "IDR",
		6:  "SEI",
		7:  "SPS",
		8:  "PPS",
		28: "FU-A",
		27: "FU-B",
		24: "STAP-A",
		0:  "RES",
	}
	var list []nalInfo
	for typ, cnt := range stats {
		desc := descs[typ]
		if desc == "" {
			desc = fmt.Sprintf("NAL%d", typ)
		}
		list = append(list, nalInfo{typ, desc, cnt})
	}
	sort.Slice(list, func(i, j int) bool { return list[i].Type < list[j].Type })
	res := ""
	for i, item := range list {
		if i > 0 {
			res += ", "
		}
		res += fmt.Sprintf("%s:%d", item.Desc, item.Cnt)
	}
	return res
}
