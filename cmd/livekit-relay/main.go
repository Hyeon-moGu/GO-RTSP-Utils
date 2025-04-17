package main

import (
    "log"
    "net"

    "github.com/pion/rtp"
    "github.com/pion/webrtc/v4"
    livekit "github.com/livekit/protocol/livekit"
    lksdk "github.com/livekit/server-sdk-go/v2"
    "go-rtsp-tools/internal/config"
)

func main() {
    config.LoadConfig()

    lk := config.Cfg.LiveKit

    room, err := lksdk.ConnectToRoom(lk.URL, lksdk.ConnectInfo{
        APIKey:              lk.APIKey,
        APISecret:           lk.APISecret,
        RoomName:            lk.RoomName,
        ParticipantIdentity: lk.Identity,
    }, nil)
    if err != nil {
        log.Fatalf("LiveKit 연결 실패: %v", err)
    }

    addr := net.UDPAddr{Port: 5004, IP: net.ParseIP("0.0.0.0")}
    conn, err := net.ListenUDP("udp", &addr)
    if err != nil {
        log.Fatalf("UDP 수신 실패: %v", err)
    }
    defer conn.Close()
    conn.SetReadBuffer(4 * 1024 * 1024)

    track, err := webrtc.NewTrackLocalStaticRTP(webrtc.RTPCodecCapability{
        MimeType:    "video/H264",
        ClockRate:   90000,
        SDPFmtpLine: "profile-level-id=42e01f;level-asymmetry-allowed=1;packetization-mode=1",
    }, "rtp-video", "stream")
    if err != nil {
        log.Fatalf("트랙 생성 실패: %v", err)
    }
    _, err = room.LocalParticipant.PublishTrack(track, &lksdk.TrackPublicationOptions{
        Name:   "rtp-video",
        Source: livekit.TrackSource_CAMERA,
    })
    if err != nil {
        log.Fatalf("트랙 퍼블리시 실패: %v", err)
    }

    buf := make([]byte, 1500)
    for {
        n, _, err := conn.ReadFromUDP(buf)
        if err != nil {
            continue
        }
        pkt := &rtp.Packet{}
        if err := pkt.Unmarshal(buf[:n]); err == nil {
            _ = track.WriteRTP(pkt)
        }
    }
}