package fmp4

import (
	"bufio"
	"fmt"
	//	"io"
	"log"
	"time"

	//	"io/ioutil"
	"os"

	"./fmp4io"
	"github.com/nareix/bits/pio"
	"github.com/nareix/joy4/av"
	"github.com/nareix/joy4/codec/aacparser"
	"github.com/nareix/joy4/codec/h264parser"
	"github.com/nareix/joy4/format/mp4/mp4io"
)

type Muxer struct {
	w             *os.File //av.PacketWriter
	maxFrames     int
	bufw          *bufio.Writer
	wpos          int64
	fragmentIndex int
	streams       []*Stream
	path          string
}

func NewMuxer(w *os.File) *Muxer {
	return &Muxer{
		w: w,
		//		bufw: bufio.NewWriterSize(w, pio.RecommendBufioSize),
		path: "null",
	}
}
func (self *Muxer) SetPath(path string) {
	self.path = path
}
func (self *Muxer) SetMaxFrames(count int) {
	self.maxFrames = count
}
func (self *Muxer) newStream(codec av.CodecData) (err error) {
	switch codec.Type() {
	case av.H264, av.AAC:

	default:
		err = fmt.Errorf("fmp4: codec type=%v is not supported", codec.Type())
		return
	}
	stream := &Stream{CodecData: codec}

	stream.sample = &mp4io.SampleTable{
		SampleDesc:    &mp4io.SampleDesc{},
		TimeToSample:  &mp4io.TimeToSample{},
		SampleToChunk: &mp4io.SampleToChunk{},
		SampleSize:    &mp4io.SampleSize{},
		ChunkOffset:   &mp4io.ChunkOffset{},
	}

	stream.trackAtom = &mp4io.Track{
		Header: &mp4io.TrackHeader{
			TrackId:  int32(len(self.streams) + 1),
			Flags:    0x0007,
			Duration: 0,
			Matrix:   [9]int32{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
		},
		Media: &mp4io.Media{
			Header: &mp4io.MediaHeader{
				TimeScale: 1000,
				Duration:  0,
				Language:  21956,
			},
			Info: &mp4io.MediaInfo{
				Sample: stream.sample,
				Data: &mp4io.DataInfo{
					Refer: &mp4io.DataRefer{
						Url: &mp4io.DataReferUrl{
							Flags: 0x000001,
						},
					},
				},
			},
		},
	}

	stream.timeScale = 48000

	switch codec.Type() {
	case av.H264:
		stream.sample.SyncSample = &mp4io.SyncSample{}
		stream.timeScale = 48000
	}

	stream.muxer = self
	self.streams = append(self.streams, stream)

	return
}

func (self *Stream) buildEsds(conf []byte) *FDummy {
	esds := &fmp4io.ElemStreamDesc{DecConfig: conf}

	b := make([]byte, esds.Len())
	esds.Marshal(b)

	esdsDummy := FDummy{
		Data: b,
		Tag_: mp4io.Tag(uint32(mp4io.ESDS)),
	}
	return &esdsDummy
}

func (self *Stream) buildHdlr() *FDummy {
	hdlr := FDummy{
		Data: []byte{
			0x00, 0x00, 0x00, 0x35, 0x68, 0x64, 0x6C, 0x72,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x76, 0x69,
			0x64, 0x65, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x47, 0x6F, 0x6F, 0x64, 0x67, 0x61,
			0x6D, 0x65, 0x20, 0x47, 0x4F, 0x20, 0x53, 0x65, 0x72, 0x76,
			0x65, 0x72, 0x00, 0x00, 0x00},

		Tag_: mp4io.Tag(uint32(mp4io.HDLR)),
	}
	return &hdlr
}

func (self *Stream) buildAudioHdlr() *FDummy {
	hdlr := FDummy{
		Data: []byte{
			0x00, 0x00, 0x00, 0x35, 0x68, 0x64, 0x6C, 0x72,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x73, 0x6F,
			0x75, 0x6E, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x42, 0x65, 0x6E, 0x74, 0x6F, 0x34,
			0x20, 0x53, 0x6F, 0x75, 0x6E, 0x64, 0x20, 0x48, 0x61, 0x6E,
			0x64, 0x6C, 0x65, 0x72, 0x00},

		Tag_: mp4io.Tag(uint32(mp4io.HDLR)),
	}
	return &hdlr
}

func (self *Stream) buildEdts() *FDummy {
	edts := FDummy{
		Data: []byte{
			0x00, 0x00, 0x00, 0x30, 0x65, 0x64, 0x74, 0x73,
			0x00, 0x00, 0x00, 0x28, 0x65, 0x6C, 0x73, 0x74, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x21,
			0xFF, 0xFF, 0xFF, 0xFF, 0x00, 0x01, 0x00, 0x00, 0x00, 0x00,
			0x9B, 0x24, 0x00, 0x00, 0x02, 0x10, 0x00, 0x01, 0x00, 0x00,
		},
		Tag_: mp4io.Tag(0x65647473),
	}
	return &edts
}

func (self *Stream) fillTrackAtom() (err error) {
	self.trackAtom.Media.Header.TimeScale = int32(self.timeScale)
	self.trackAtom.Media.Header.Duration = int32(self.duration)

	//log.Print("track Type: ", self.Type())

	if self.Type() == av.H264 {
		codec := self.CodecData.(h264parser.CodecData)
		width, height := codec.Width(), codec.Height()
		self.sample.SampleDesc.AVC1Desc = &mp4io.AVC1Desc{
			DataRefIdx:           1,
			HorizontalResolution: 72,
			VorizontalResolution: 72,
			Width:                int16(width),
			Height:               int16(height),
			FrameCount:           1,
			Depth:                24,
			ColorTableId:         -1,
			Conf:                 &mp4io.AVC1Conf{Data: codec.AVCDecoderConfRecordBytes()},
		}
		self.trackAtom.Header.TrackWidth = float64(width)
		self.trackAtom.Header.TrackHeight = float64(height)

		self.trackAtom.Media.Handler = &mp4io.HandlerRefer{
			SubType: [4]byte{'v', 'i', 'd', 'e'},
			Name:    []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 'G', 'G', 0, 0, 0},
		}

		//self.trackAtom.Media.Unknowns = []mp4io.Atom{self.buildHdlr()}
		self.trackAtom.Media.Info.Video = &mp4io.VideoMediaInfo{
			Flags: 0x000001,
		}

		self.codecString = fmt.Sprintf("avc1.%02X%02X%02X", codec.RecordInfo.AVCProfileIndication, codec.RecordInfo.ProfileCompatibility, codec.RecordInfo.AVCLevelIndication)

	} else if self.Type() == av.AAC {
		codec := self.CodecData.(aacparser.CodecData)

		self.sample.SampleDesc.MP4ADesc = &mp4io.MP4ADesc{
			DataRefIdx:       1,
			NumberOfChannels: int16(codec.ChannelLayout().Count()),
			SampleSize:       16, //int16(codec.SampleFormat().BytesPerSample()),
			SampleRate:       float64(codec.SampleRate()),
			//Conf: &mp4io.ElemStreamDesc{
			//	DecConfig: codec.MPEG4AudioConfigBytes(),
			//},
			Unknowns: []mp4io.Atom{self.buildEsds(codec.MPEG4AudioConfigBytes())},
		}
		self.trackAtom.Header.Volume = 1
		self.trackAtom.Header.AlternateGroup = 1
		self.trackAtom.Header.Duration = 0

		self.trackAtom.Media.Handler = &mp4io.HandlerRefer{
			SubType: [4]byte{'s', 'o', 'u', 'n'},
			Name:    []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 'G', 'G', 0, 0, 0},
		}

		self.trackAtom.Media.Info.Sound = &mp4io.SoundMediaInfo{}
		self.codecString = "mp4a.40.2"

	} else {
		err = fmt.Errorf("fmp4: codec type=%d invalid", self.Type())
	}

	return
}

func (self *Muxer) WriteTrailer() (err error) {
	return
}

func (self *Muxer) WriteHeader(streams []av.CodecData, WriteInit bool) (err error) {
	self.streams = []*Stream{}
	for _, stream := range streams {
		if err = self.newStream(stream); err != nil {
			return
		}
	}
	if WriteInit {
		if err = self.WriteInit(self.path); err != nil {
			return
		}
	}

	return
}

func (self *Muxer) WritePacket(pkt av.Packet) (err error) {
	//log.Print("Packet: ", pkt.Idx)

	if len(self.streams) > int(pkt.Idx) {
		stream := self.streams[pkt.Idx]
		if stream.lastpkt != nil {
			if err = stream.writePacket(*stream.lastpkt, pkt.Time-stream.lastpkt.Time, self.maxFrames); err != nil {
				return
			}
		} else {
			if stream.CodecData.Type() == av.H264 {
				println("First packet is keyframe? ", pkt.IsKeyFrame)
			}
		}
		stream.lastpkt = &pkt
	}

	return
}

// TODO: Переписать, костыли!
func (self *Stream) writePacket(pkt av.Packet, rawdur time.Duration, maxFrames int) (err error) {
	if rawdur < 0 {
		err = fmt.Errorf("fmp4: stream#%d time=%v < lasttime=%v", pkt.Idx, pkt.Time, self.lastpkt.Time)
		return
	}

	//if self.CodecData.Type() == av.H264 {
	//log.Print("Packet ", self.sampleIndex, pkt.IsKeyFrame)
	//	os.Exit(0)
	//}

	if self.CodecData.Type() == av.AAC {
		maxFrames = maxFrames * 5
	}

	trackId := pkt.Idx + 1

	//if self.CodecData.Type() == av.H264 {
	//	log.Print("Pkt: ", self.sampleIndex, pkt.IsKeyFrame)
	//}

	if self.sampleIndex == 0 {
		//if self.CodecData.Type() == av.H264 {
		//	log.Print(trackId, pkt.IsKeyFrame)
		//}

		self.moof.Header = &fmp4io.MovieFragHeader{Seqnum: uint32(self.muxer.fragmentIndex + 1)}
		self.moof.Tracks = []*fmp4io.TrackFrag{
			&fmp4io.TrackFrag{
				Header: &fmp4io.TrackFragHeader{
					Data: []byte{0x00, 0x02, 0x00, 0x20, 0x00, 0x00, 0x00, uint8(trackId), 0x01, 0x01, 0x00, 0x00},
				},
				DecodeTime: &fmp4io.TrackFragDecodeTime{
					Version: 1,
					Flags:   0,
					Time:    uint64(self.dts),
				},
				Run: &fmp4io.TrackFragRun{
					Flags:            0x000b05,
					FirstSampleFlags: 0x02000000,
					DataOffset:       0,
					Entries:          []mp4io.TrackFragRunEntry{},
				},
			},
		}

		self.buffer = []byte{0x00, 0x00, 0x00, 0x00, 0x6d, 0x64, 0x61, 0x74}
	}

	runEnrty := mp4io.TrackFragRunEntry{
		Duration: uint32(self.timeToTs(rawdur)),
		Size:     uint32(len(pkt.Data)),
		Cts:      uint32(self.timeToTs(pkt.CompositionTime)),
	}
	self.moof.Tracks[0].Run.Entries = append(self.moof.Tracks[0].Run.Entries, runEnrty)
	self.buffer = append(self.buffer, pkt.Data...)

	self.sampleIndex++
	self.dts += self.timeToTs(rawdur)
	//fmt.Println(self.dts)

	if self.sampleIndex > maxFrames { // Количество фреймов в пакете
		self.moof.Tracks[0].Run.DataOffset = uint32(self.moof.Len() + 8)

		file := make([]byte, self.moof.Len()+len(self.buffer))

		//moofData := make([]byte, self.moof.Len())
		self.moof.Marshal(file)

		pio.PutU32BE(self.buffer, uint32(len(self.buffer)))

		copy(file[self.moof.Len():], self.buffer)

		//file := make([]byte, 0)
		//file = append(file, moofData...) // TODO: Переделать на нормальную запись
		//file = append(file, self.buffer...)

		// Сохраняем сегменты для дебага:
		//log.Print("Saving segment ", self.muxer.fragmentIndex+1)
		//ioutil.WriteFile(fmt.Sprintf("res/%d.m4s", self.muxer.fragmentIndex+1), file, 0644)

		self.sampleIndex = 0
		self.muxer.fragmentIndex++

		cfp := av.Packet{
			IsKeyFrame:      pkt.IsKeyFrame,
			CompositionTime: pkt.CompositionTime,
			Idx:             pkt.Idx,
			Time:            pkt.Time,
			Data:            file,
		}
		//fmt.Println(cfp)
		self.muxer.w.Write(cfp.Data)
		//self.muxer.w.WriteString(cfp.Data)
		//		self.muxer.w.WritePacket(cfp)

		/*		if _, err = self.muxer.w.Write(cfp); err != nil {
					log.Print(err)
					return
				}

				//if trackId == 3 {
				//	os.Exit(0)
				//}*/
	}

	return
}

func (self *Muxer) buildMvex() *FDummy {
	mvex := &FDummy{
		Data: []byte{
			0x00, 0x00, 0x00, 0x38, 0x6D, 0x76, 0x65, 0x78,
			0x00, 0x00, 0x00, 0x10, 0x6D, 0x65, 0x68, 0x64, 0x00, 0x00,
			0x00, 0x00, 0x00, 0x00, 0x00, 0x00},
		Tag_: mp4io.Tag(0x6D766578),
	}

	for i := 1; i <= len(self.streams); i++ {
		trex := self.buildTrex(i)
		mvex.Data = append(mvex.Data, trex...)
	}

	pio.PutU32BE(mvex.Data, uint32(len(mvex.Data)))
	return mvex
}

func (self *Muxer) buildTrex(trackId int) []byte {
	return []byte{
		0x00, 0x00, 0x00, 0x20, 0x74, 0x72, 0x65, 0x78,
		0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, uint8(trackId), 0x00, 0x00,
		0x00, 0x01, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00,
		0x00, 0x00, 0x00, 0x00}
}

func (self *Muxer) WriteInit(path string) (err error) {

	moov := &mp4io.Movie{
		Header: &mp4io.MovieHeader{
			PreferredRate:   1,
			PreferredVolume: 1,
			Matrix:          [9]int32{0x10000, 0, 0, 0, 0x10000, 0, 0, 0, 0x40000000},
			NextTrackId:     -1,
			Duration:        0,
			TimeScale:       1000,

			CreateTime:        time0(),
			ModifyTime:        time0(),
			PreviewTime:       time0(),
			PreviewDuration:   time0(),
			PosterTime:        time0(),
			SelectionTime:     time0(),
			SelectionDuration: time0(),
			CurrentTime:       time0(),
		},
		Unknowns: []mp4io.Atom{self.buildMvex()},
	}

	for _, stream := range self.streams {
		if err = stream.fillTrackAtom(); err != nil {
			return
		}
		moov.Tracks = append(moov.Tracks, stream.trackAtom)
	}

	ftypeData := []byte{
		0x00, 0x00, 0x00, 0x24,
		0x66, 0x74, 0x79, 0x70, 0x69, 0x73, 0x6F, 0x6D,
		0x00, 0x00, 0x02, 0x00, 0x69, 0x73, 0x6F, 0x6D,
		0x69, 0x73, 0x6F, 0x32, 0x61, 0x76, 0x63, 0x31,
		0x6D, 0x70, 0x34, 0x31, 0x69, 0x73, 0x6F, 0x35}

	file := make([]byte, moov.Len()+len(ftypeData))
	copy(file, ftypeData)
	moov.Marshal(file[len(ftypeData):])

	//ioutil.WriteFile("res/init.mp4", file, 0644)
	//Path to init
	var pcs av.Packet
	pcs.Data = file
	//self.w.WritePacket(pcs)
	if path != "null" {
		self.w.Write(file)
		//ioutil.WriteFile(path, file, 0644)
	}
	// INIT не пишем в сокет!
	//if _, err = self.w.Write(file); err != nil {
	//	log.Print("error: ", err)
	//	return
	//}

	log.Print("done init")

	return
}

type FDummy struct {
	Data []byte
	Tag_ mp4io.Tag
	mp4io.AtomPos
}

func (self FDummy) Children() []mp4io.Atom {
	return nil
}

func (self FDummy) Tag() mp4io.Tag {
	return self.Tag_
}

func (self FDummy) Len() int {
	return len(self.Data)
}

func (self FDummy) Marshal(b []byte) int {
	copy(b, self.Data)
	return len(self.Data)
}

func (self FDummy) Unmarshal(b []byte, offset int) (n int, err error) {
	return
}

func time0() time.Time {
	return time.Date(1904, time.January, 1, 0, 0, 0, 0, time.UTC)
}
