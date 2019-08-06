package ts

import (
	"fmt"
	"io"

	//	"io/ioutil"
	"os"
	"strconv"

	//	"strings"
	"time"

	"Recorder/joy4/av"
	"Recorder/joy4/codec/aacparser"
	"Recorder/joy4/codec/h264parser"
	"Recorder/joy4/format/ts/tsio"
)

var CodecTypes = []av.CodecType{av.H264, av.AAC}

//var chunk *os.File
//var playlist *os.File
//var packetCounter = 0
//var chunkCounter = 0

type Muxer struct {
	w                        io.Writer
	streams                  []*Stream
	PaddingToMakeCounterCont bool

	psidata        []byte
	peshdr         []byte
	tshdr          []byte
	adtshdr        []byte
	datav          [][]byte
	nalus          [][]byte
	strName        string
	path           string
	tswpat, tswpmt *tsio.TSWriter
	chunkCounter   int
	packetCounter  int
	playlist       *os.File
	chunk          *os.File
}

func NewMuxer(streamName string, path string, w io.WriteSeeker) *Muxer {
	//chunk, _ = io.Writer //os.Create(path + "/" + streamName + "-" + strconv.Itoa(chunkCounter) + ".ts")

	//	playlist.WriteString(streamName + "-" + strconv.Itoa(chunkCounter) + ".ts\n")
	return &Muxer{
		w:             w,
		psidata:       make([]byte, 188),
		peshdr:        make([]byte, tsio.MaxPESHeaderLength),
		tshdr:         make([]byte, tsio.MaxTSHeaderLength),
		adtshdr:       make([]byte, aacparser.ADTSHeaderLength),
		nalus:         make([][]byte, 16),
		datav:         make([][]byte, 16),
		tswpmt:        tsio.NewTSWriter(tsio.PMT_PID),
		tswpat:        tsio.NewTSWriter(tsio.PAT_PID),
		strName:       streamName,
		chunkCounter:  0,
		packetCounter: 0,
		playlist:      createFile(path + "/" + streamName + ".m3u8"),
		chunk:         createFile(path + "/" + streamName + "-0.ts"),
		path:          path,
	}
}
func createFile(str string) *os.File {
	file, _ := os.Create(str)
	return file
}
func (self *Muxer) WritePlaylist() {
	self.playlist.WriteString("#EXTM3U\n")
	self.playlist.WriteString("#EXT-X-VERSION:3\n")
	self.playlist.WriteString("#EXT-X-MEDIA-SEQUENCE:1\n")
	self.playlist.WriteString("#EXT-X-TARGETDURATION:2\n")
	self.playlist.WriteString("#EXTINF:2.000,\n")
	self.playlist.WriteString(self.strName + "-0.ts\n")
}
func (self *Muxer) newStream(codec av.CodecData) (err error) {
	ok := false
	for _, c := range CodecTypes {
		if codec.Type() == c {
			ok = true
			break
		}
	}
	if !ok {
		err = fmt.Errorf("ts: codec type=%s is not supported", codec.Type())
		return
	}

	pid := uint16(len(self.streams) + 0x100)
	stream := &Stream{
		muxer:     self,
		CodecData: codec,
		pid:       pid,
		tsw:       tsio.NewTSWriter(pid),
	}
	self.streams = append(self.streams, stream)
	return
}

func (self *Muxer) writePaddingTSPackets(tsw *tsio.TSWriter) (err error) {
	for tsw.ContinuityCounter&0xf != 0x0 {
		if err = tsw.WritePackets(self.w, self.datav[:0], 0, false, true); err != nil {
			return
		}
	}
	return
}

func (self *Muxer) WriteTrailer() (err error) {
	if self.PaddingToMakeCounterCont {
		for _, stream := range self.streams {
			if err = self.writePaddingTSPackets(stream.tsw); err != nil {
				return
			}
		}
	}
	return
}

func (self *Muxer) SetWriter(w io.Writer) {
	self.w = w
	return
}

func (self *Muxer) WritePATPMT() (err error) {
	pat := tsio.PAT{
		Entries: []tsio.PATEntry{
			{ProgramNumber: 1, ProgramMapPID: tsio.PMT_PID},
		},
	}
	patlen := pat.Marshal(self.psidata[tsio.PSIHeaderLength:])
	n := tsio.FillPSI(self.psidata, tsio.TableIdPAT, tsio.TableExtPAT, patlen)
	self.datav[0] = self.psidata[:n]
	if err = self.tswpat.WritePackets(self.w, self.datav[:1], 0, false, true); err != nil {
		return
	}

	var elemStreams []tsio.ElementaryStreamInfo
	for _, stream := range self.streams {
		switch stream.Type() {
		case av.AAC:
			elemStreams = append(elemStreams, tsio.ElementaryStreamInfo{
				StreamType:    tsio.ElementaryStreamTypeAdtsAAC,
				ElementaryPID: stream.pid,
			})
		case av.H264:
			elemStreams = append(elemStreams, tsio.ElementaryStreamInfo{
				StreamType:    tsio.ElementaryStreamTypeH264,
				ElementaryPID: stream.pid,
			})
		}
	}

	pmt := tsio.PMT{
		PCRPID:                0x100,
		ElementaryStreamInfos: elemStreams,
	}
	pmtlen := pmt.Len()
	if pmtlen+tsio.PSIHeaderLength > len(self.psidata) {
		err = fmt.Errorf("ts: pmt too large")
		return
	}
	pmt.Marshal(self.psidata[tsio.PSIHeaderLength:])
	n = tsio.FillPSI(self.psidata, tsio.TableIdPMT, tsio.TableExtPMT, pmtlen)
	self.datav[0] = self.psidata[:n]
	if err = self.tswpmt.WritePackets(self.w, self.datav[:1], 0, false, true); err != nil {
		return
	}

	return
}

func (self *Muxer) WriteHeader(streams []av.CodecData) (err error) {
	self.streams = []*Stream{}
	for _, stream := range streams {
		if err = self.newStream(stream); err != nil {
			return
		}
	}

	if err = self.WritePATPMT(); err != nil {
		return
	}
	return
}

func (self *Muxer) WritePacket(pkt av.Packet) (err error) {
	//fmt.Println(self.packetCounter)
	//pkt.IsKeyFrame
	if /*self.packetCounter == 300*/ pkt.IsKeyFrame {
		self.chunkCounter += 1
		self.chunk.Close()
		self.playlist.WriteString("#EXTINF:2.000,")
		/*input, _ := ioutil.ReadFile(self.path + "/" + self.strName + ".m3u8")
		lines := strings.Split(string(input), "\n")
		for i, line := range lines {
			if strings.HasPrefix(line, "#EXT-X-MEDIA-SEQUENCE:") {
				lines[i] = "#EXT-X-MEDIA-SEQUENCE:" + strconv.Itoa(self.chunkCounter)
			}
		}
		output := strings.Join(lines, "\n")*/
		//		ioutil.WriteFile(self.path+"/"+self.strName+".m3u8", []byte(output), 0644)
		self.playlist.WriteString("\n" + self.strName + "-" + strconv.Itoa(self.chunkCounter) + ".ts\n")
		self.chunk, _ = os.Create(self.path + "/" + self.strName + "-" + strconv.Itoa(self.chunkCounter) + ".ts")
		//fmt.Println(self.path + "/" + self.strName + "-" + strconv.Itoa(self.chunkCounter) + ".ts")
		self.w = self.chunk
		if err = self.WritePATPMT(); err != nil {
			return
		}
		self.packetCounter = 0
	}
	self.packetCounter += 1
	stream := self.streams[pkt.Idx]
	pkt.Time += time.Second

	switch stream.Type() {
	case av.AAC:
		codec := stream.CodecData.(aacparser.CodecData)

		n := tsio.FillPESHeader(self.peshdr, tsio.StreamIdAAC, len(self.adtshdr)+len(pkt.Data), pkt.Time, 0)
		self.datav[0] = self.peshdr[:n]
		aacparser.FillADTSHeader(self.adtshdr, codec.Config, 1024, len(pkt.Data))
		self.datav[1] = self.adtshdr
		self.datav[2] = pkt.Data

		if err = stream.tsw.WritePackets(self.w, self.datav[:3], pkt.Time, true, false); err != nil {
			return
		}

	case av.H264:
		codec := stream.CodecData.(h264parser.CodecData)

		nalus := self.nalus[:0]
		if pkt.IsKeyFrame {
			nalus = append(nalus, codec.SPS())
			nalus = append(nalus, codec.PPS())
		}
		pktnalus, _ := h264parser.SplitNALUs(pkt.Data)
		for _, nalu := range pktnalus {
			nalus = append(nalus, nalu)
		}

		datav := self.datav[:1]
		for i, nalu := range nalus {
			if i == 0 {
				datav = append(datav, h264parser.AUDBytes)
			} else {
				datav = append(datav, h264parser.StartCodeBytes)
			}
			datav = append(datav, nalu)
		}

		n := tsio.FillPESHeader(self.peshdr, tsio.StreamIdH264, -1, pkt.Time+pkt.CompositionTime, pkt.Time)
		datav[0] = self.peshdr[:n]

		if err = stream.tsw.WritePackets(self.w, datav, pkt.Time, pkt.IsKeyFrame, false); err != nil {
			return
		}
	}

	return
}
