package main

import (
    "io"
    "os"
    "fmt"
    "errors"
    "strconv"
    "container/list"
)

const MAX_NAL_UNITS_PER_BS = 600
const BITSTREAM_BUFFER_SIZE = 1024*1024*8
const START_CODE_SIZE  = 3

type sBitstream struct {
    nBufSize            uint32;
    pData               []byte;
    anNalUnitType       [MAX_NAL_UNITS_PER_BS+1]uint8;
    anNalUnitLocation   [MAX_NAL_UNITS_PER_BS+1]uint32;
    nNumNalUnits        uint32;
    bAccessUnitIDRFlag  bool
};

func FindAuNalUnits(pBs *sBitstream, fpBs io.ReadSeeker) (bool, error){
    var buffer [1]byte
    var nBs []byte;
    var pData []byte;
    var nBsSize uint32;
    var pNalUnitType        []uint8;
    var pNalUnitLocation    []uint32;
    var nNumNalUnits    uint32;
    var nZeros  uint32;
    var nNalUnitType    byte;
    var bFirstSliceInPicFlag    bool;
    var bPicFoundFlag   bool;
    var nBsSizeSinceLastSlice   uint32;
    var nNumNalUnitsSinceLastSlice  uint32;
    var bLastSliceFlag  bool;

    nBs = buffer[:]
    pData = pBs.pData;
    nBsSize = 0;
    pNalUnitType     = pBs.anNalUnitType[:];
    pNalUnitLocation = pBs.anNalUnitLocation[:];
    nNumNalUnits = 0;

    nZeros = 0;
    bPicFoundFlag = false;
    nBsSizeSinceLastSlice = START_CODE_SIZE;
    nNumNalUnitsSinceLastSlice = 0;
    bLastSliceFlag = true;

    for nBsSize < pBs.nBufSize {
        if n, err := fpBs.Read(nBs); n!=1 {
            if err == io.EOF {
                pNalUnitLocation[nNumNalUnits] = nBsSize;
                pBs.nNumNalUnits = nNumNalUnits;
                return true, err;
            }else {
                return false,err;
            }
        }
        pData[nBsSize] = nBs[0];
        nBsSize++;
        if false == bLastSliceFlag {
            nBsSizeSinceLastSlice++;
        }

        switch nBs[0] {
            case 0:
                nZeros++;
            case 1:
                if nZeros > 1 { // find trailing_zero_8bits and 0x000001
                    pNalUnitLocation[nNumNalUnits] = nBsSize-nZeros-1;

                    if n, err := fpBs.Read(nBs); n!=1 {
                        if err == io.EOF {
                            return true,err
                        }else {
                            return false,err
                        }
                    }

                    nNalUnitType = (nBs[0]&0x7E)>>1;

                    pNalUnitType[nNumNalUnits] = nNalUnitType;

                    if nNalUnitType <=23{ // SLICE FOUND
                        if n, err := fpBs.Read(nBs); n!=1 {
                            if err == io.EOF {
                                return true, err;
                            }else {
                                return false, err;
                            }
                        }
                        if n, err := fpBs.Read(nBs); n!=1 {
                            if err == io.EOF {
                                return true, err;
                            }else {
                                return false, err;
                            }
                        }

                        bFirstSliceInPicFlag = (nBs[0]>>7)!=0;

                        if bFirstSliceInPicFlag {
                            if bPicFoundFlag {
                                fpBs.Seek(-3-int64(nBsSizeSinceLastSlice), 1);
                                pBs.nNumNalUnits = nNumNalUnits-nNumNalUnitsSinceLastSlice;

                                return true, nil;
                            }else {
                                fpBs.Seek(-3, 1);
                                nNumNalUnits++;
                                nZeros = 0;
                                bPicFoundFlag = true;
                                pBs.bAccessUnitIDRFlag = (nNalUnitType==19 /*NAL_UNIT_CODED_SLICE_IDR*/ || nNalUnitType == 20 /*NAL_UNIT_CODED_SLICE_IDR_N_LP*/);
                            }
                        }else {
                            fpBs.Seek(-3, 1);
                            nNumNalUnits++;
                            nZeros = 0;
                        }

                        bLastSliceFlag = true;
                        nBsSizeSinceLastSlice = START_CODE_SIZE;
                        nNumNalUnitsSinceLastSlice = 0;
                    }else {
                        if nNalUnitType==40{//Suffix SEI
                            bLastSliceFlag = true;
                            nBsSizeSinceLastSlice = START_CODE_SIZE;
                            nNumNalUnitsSinceLastSlice = 0;
                        }else{
                            bLastSliceFlag = false;
                            nNumNalUnitsSinceLastSlice++;
                        }

                        fpBs.Seek(-1, 1);
                        nNumNalUnits++;
                        nZeros = 0;
                    }
                }else {
                    nZeros = 0;
                }
            default:
                nZeros = 0;
        }
    }

    return false, errors.New("nBsSize exceed pBs.nBufSize\n");
}

func main() {
    if len(os.Args)<4 {
        print("Usage: GoSplit.exe input.bin output cutFrameNum");
        return;
    }

    cutFrameNum, err0 := strconv.Atoi(os.Args[3]);
    if err0!=nil {
        print("cut_framenum is not digital");
        return;
    }

    bitstreamFile, err := os.Open(os.Args[1])
    if err != nil {
        fmt.Printf("\nFailed to open bitstream file %s\n", os.Args[1])
        return
    }
    defer bitstreamFile.Close()

    var sBs  sBitstream;

    sBs.pData = make([]byte, BITSTREAM_BUFFER_SIZE);
    sBs.nBufSize = BITSTREAM_BUFFER_SIZE;

    var outputName string
    var outputFile *os.File
    var psList *list.List

    psList = list.New();
    prevFrameNo:=0;
    nFrames:= 0;
    nIDR:= 0;
    ret := false;
    err  = nil
    for err!=io.EOF {
	    sBs.nBufSize = BITSTREAM_BUFFER_SIZE;
        if ret, err = FindAuNalUnits(&sBs, bitstreamFile); ret==false {
            print("FindAuNalUints Failed!");
            return;
        }
		sBs.nBufSize = sBs.anNalUnitLocation[sBs.nNumNalUnits] - sBs.anNalUnitLocation[0];

        if sBs.bAccessUnitIDRFlag && (nFrames-prevFrameNo >= cutFrameNum||nIDR==0) {
            if nIDR>0 {
                fmt.Printf("\nFrames[%04d-%04d] => %s\n", prevFrameNo, nFrames-1, outputName);
                outputFile.Close();
            }
            prevFrameNo = nFrames;
            outputName = fmt.Sprintf("%s_%04d.bin", os.Args[2], prevFrameNo);
            nIDR++;

            outputFile, err = os.Create(outputName);
            if err != nil {
                fmt.Printf("\nFailed to create output file %s\n", outputName)
                return
            }

            for e:=psList.Front(); e!=nil; e=e.Next() {
                psData := e.Value.([]byte);
                outputFile.Write(psData[:]);
            }

            outputFile.Write(sBs.pData[:sBs.nBufSize]);

        }else{
            if outputFile!=nil {
                outputFile.Write(sBs.pData[:sBs.nBufSize]);
            }

        }
        if sBs.bAccessUnitIDRFlag {
            fmt.Printf("IDR");
        }else{
            fmt.Printf(".");
        }

        for i:=uint32(0); i<sBs.nNumNalUnits; i++ {
            if sBs.anNalUnitType[i]==32 || /*NAL_UNIT_VPS*/
               sBs.anNalUnitType[i]==33 || /*NAL_UNIT_SPS*/
               sBs.anNalUnitType[i]==34    /*NAL_UNIT_PPS*/  {
                psData := make([]byte, sBs.anNalUnitLocation[i+1] - sBs.anNalUnitLocation[i]);
                for j:=uint32(0); j<sBs.anNalUnitLocation[i+1] - sBs.anNalUnitLocation[i]; j++ {
                    psData[j] =  sBs.pData[sBs.anNalUnitLocation[i]+j];
                }
                psList.PushBack(psData);
            }
        }

        nFrames++;
    }

    if nIDR>0 {
        fmt.Printf("\nFrames[%04d-%04d] => %s\n", prevFrameNo, nFrames-1, outputName);
        outputFile.Close();

        fmt.Printf("\nSummary: Total Frames %d, IDR Groups %d\n", nFrames, nIDR);
        fmt.Printf("SubStreams: %s_[%03d-%03d].bin are extracted!\n", os.Args[2], 0, nIDR-1);
    }else{
        fmt.Printf("No IDR or SubStreasm are extracted!\n")
    }
}
