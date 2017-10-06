package main

import (
	"container/list"
	"fmt"
	"io"
	"os"
	"strconv"
)

const MAX_NAL_UNITS_PER_BS = 600
const BITSTREAM_BUFFER_SIZE = 1024 * 1024 * 8
const START_CODE_SIZE = 3

type sBitstream struct {
	frame_header       []byte
	nBufSize           uint32
	pData              []byte
	bAccessUnitIDRFlag bool
}

func FindAuNalUnits(pBs *sBitstream, fpBs io.ReadSeeker, isVP9 int) error {
	var err error
	var read_buffer_size int

	pBs.frame_header = make([]byte, 12)
	read_buffer_size, err = fpBs.Read(pBs.frame_header[:])
	if read_buffer_size != 12 {
		return err
	}

	//bytes 0-3    size of frame in bytes (not including the 12-byte header)
	pBs.nBufSize = (uint32(pBs.frame_header[3]) << 24) | (uint32(pBs.frame_header[2]) << 16) |
		(uint32(pBs.frame_header[1]) << 8) | (uint32(pBs.frame_header[0]) << 0)

	pBs.pData = make([]byte, pBs.nBufSize)
	read_buffer_size, err = fpBs.Read(pBs.pData[:])
	if read_buffer_size != int(pBs.nBufSize) {
		return err
	}

	if isVP9 == 0 { //VP8
		key_frame := (pBs.pData[0] & 0x1)
		if key_frame == 0 {
			pBs.bAccessUnitIDRFlag = true
		} else {
			pBs.bAccessUnitIDRFlag = false
		}
	} else { //VP9
		show_existing_frame := (pBs.pData[0] & 0x8)

		key_frame := (pBs.pData[0] & 0x4)
		if key_frame == 0 && show_existing_frame == 0 {
			pBs.bAccessUnitIDRFlag = true
		} else {
			pBs.bAccessUnitIDRFlag = false
		}
	}

	return nil
}

func main() {
	if len(os.Args) < 5 {
		print("Usage: GoSplitIVF.exe input.ivf output cutFrameNum isVP9")
		return
	}

	cutFrameNum, err0 := strconv.Atoi(os.Args[3])
	if err0 != nil {
		print("cut_framenum is not digital")
		return
	}

	isVP9, err1 := strconv.Atoi(os.Args[4])
	if err1 != nil {
		print("isVP9 is not digitial")
		return
	}

	bitstreamFile, err := os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("\nFailed to open bitstream file %s\n", os.Args[1])
		return
	}
	defer bitstreamFile.Close()

	var sBs sBitstream
	var outputName string
	var outputFile *os.File
	var psList *list.List

	psList = list.New()
	curFrameNum := 0
	prevFrameNo := 0
	nFrames := 0
	nIDR := 0

	var m_ivf_buffer [32]byte
	var read_buffer_size int
	read_buffer_size, err = bitstreamFile.Read(m_ivf_buffer[:])
	if read_buffer_size != 32 {
		print("Failed to read ivf!")
		return
	}

	//bytes 24-27  number of frames in file
	totalFrameNum := (uint(m_ivf_buffer[27]) << 24) | (uint(m_ivf_buffer[26]) << 16) |
		(uint(m_ivf_buffer[25]) << 8) | (uint(m_ivf_buffer[24]) << 0)

	if totalFrameNum == 0 {
		totalFrameNum = 10000
	}

	for frameNo := uint(0); frameNo < totalFrameNum; frameNo++ {
		if err = FindAuNalUnits(&sBs, bitstreamFile, isVP9); err != nil {
			break
		}

		if sBs.bAccessUnitIDRFlag && (nFrames-prevFrameNo >= cutFrameNum || nIDR == 0) {
			if nIDR > 0 {
				fmt.Printf("\nFrames[%04d-%04d] => %s\n", prevFrameNo, nFrames-1, outputName)
				curFrameNum = nFrames - prevFrameNo
				m_ivf_buffer[24] = byte((curFrameNum >> 0) & 0xFF)
				m_ivf_buffer[25] = byte((curFrameNum >> 8) & 0xFF)
				m_ivf_buffer[26] = byte((curFrameNum >> 16) & 0xFF)
				m_ivf_buffer[27] = byte((curFrameNum >> 24) & 0xFF)
				outputFile.Write(m_ivf_buffer[:])
				for e := psList.Front(); e != nil; e = e.Next() {
					psData := e.Value.([]byte)
					outputFile.Write(psData[:])
				}
				outputFile.Close()
				psList.Init()
				if psList.Len() != 0 {
					print("Failed to remove psList")
				}
			}
			prevFrameNo = nFrames
			outputName = fmt.Sprintf("%s_%04d.ivf", os.Args[2], prevFrameNo)
			nIDR++

			outputFile, err = os.Create(outputName)
			if err != nil {
				fmt.Printf("\nFailed to create output file %s\n", outputName)
				return
			}

			psList.PushBack(sBs.frame_header)
			psList.PushBack(sBs.pData)
			//outputFile.Write(sBs.frame_header[:])
			//outputFile.Write(sBs.pData[:sBs.nBufSize])
		} else {
			if outputFile != nil {
				psList.PushBack(sBs.frame_header)
				psList.PushBack(sBs.pData)
				//outputFile.Write(sBs.frame_header[:])
				//outputFile.Write(sBs.pData[:sBs.nBufSize])
			}

		}
		if sBs.bAccessUnitIDRFlag {
			fmt.Printf("IDR")
		} else {
			fmt.Printf(".")
		}

		nFrames++
	}

	if nIDR > 0 {
		fmt.Printf("\nFrames[%04d-%04d] => %s\n", prevFrameNo, nFrames-1, outputName)
		curFrameNum = nFrames - prevFrameNo
		m_ivf_buffer[24] = byte((curFrameNum >> 0) & 0xFF)
		m_ivf_buffer[25] = byte((curFrameNum >> 8) & 0xFF)
		m_ivf_buffer[26] = byte((curFrameNum >> 16) & 0xFF)
		m_ivf_buffer[27] = byte((curFrameNum >> 24) & 0xFF)
		outputFile.Write(m_ivf_buffer[:])
		for e := psList.Front(); e != nil; e = e.Next() {
			psData := e.Value.([]byte)
			outputFile.Write(psData[:])
		}
		outputFile.Close()
		psList.Init()
		if psList.Len() != 0 {
			print("Failed to remove psList")
		}

		fmt.Printf("\nSummary: Total Frames %d, IDR Groups %d\n", nFrames, nIDR)
		fmt.Printf("SubStreams: %s_[%03d-%03d].ivf are extracted!\n", os.Args[2], 0, nIDR-1)
	} else {
		fmt.Printf("No Key-Frame or SubStreasm are extracted!\n")
	}
}
