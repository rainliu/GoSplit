package main

import (
	"fmt"
	"math"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 6 {
		print("Usage: GoPSNR.exe origin.yuv recon.yuv width height framenum")
		return
	}
	var err error
	var width, height, framenum int
	if width, err = strconv.Atoi(os.Args[3]); err != nil {
		print("width is not digital")
		return
	}
	if height, err = strconv.Atoi(os.Args[4]); err != nil {
		print("height is not digital")
		return
	}
	if framenum, err = strconv.Atoi(os.Args[5]); err != nil {
		print("framenum is not digital")
		return
	}

	var inputFile, outputFile *os.File

	inputFile, err = os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("\nFailed to open origin file %s\n", os.Args[1])
		return
	}
	defer inputFile.Close()

	outputFile, err = os.Open(os.Args[2])
	if err != nil {
		fmt.Printf("\nFailed to create recon file %s\n", os.Args[2])
		return
	}
	defer outputFile.Close()

	var read_buffer_size int
	buffer_size := (width * height * 3) / 2
	buffer_org := make([]byte, buffer_size)
	buffer_rec := make([]byte, buffer_size)

	TOTAL_PSNR_Y := float64(0.0)
	TOTAL_PSNR_U := float64(0.0)
	TOTAL_PSNR_V := float64(0.0)
	TOTAL_PSNR := float64(0.0)
	for i := 0; i < framenum; i++ {
		if read_buffer_size, err = inputFile.Read(buffer_org); err != nil || read_buffer_size != buffer_size {
			fmt.Printf("\nFailed to read origin file %s\n", os.Args[1])
			return
		}
		if read_buffer_size, err = outputFile.Read(buffer_rec); err != nil || read_buffer_size != buffer_size {
			fmt.Printf("\nFailed to read origin file %s\n", os.Args[2])
			return
		}

		MSE_Y := float64(0.0)
		MSE_U := float64(0.0)
		MSE_V := float64(0.0)

		for n := 0; n < height; n++ {
			for m := 0; m < width; m++ {
				MSE_Y += (float64(buffer_org[n*width+m]) - float64(buffer_rec[n*width+m])) *
					(float64(buffer_org[n*width+m]) - float64(buffer_rec[n*width+m]))
			}
		}

		for n := 0; n < height/2; n++ {
			for m := 0; m < width/2; m++ {
				MSE_U += (float64(buffer_org[width*height+n*width/2+m]) - float64(buffer_rec[width*height+n*width/2+m])) *
					(float64(buffer_org[width*height+n*width/2+m]) - float64(buffer_rec[width*height+n*width/2+m]))
				MSE_V += (float64(buffer_org[width*height+width*height/4+n*width/2+m]) - float64(buffer_rec[width*height+width*height/4+n*width/2+m])) *
					(float64(buffer_org[width*height+width*height/4+n*width/2+m]) - float64(buffer_rec[width*height+width*height/4+n*width/2+m]))
			}
		}

		MSE_Y /= float64(width * height)
		MSE_U /= float64((width * height) / 4)
		MSE_V /= float64((width * height) / 4)

		PSNR_Y := 10 * math.Log10(float64(255*255)/MSE_Y)
		PSNR_U := 10 * math.Log10(float64(255*255)/MSE_U)
		PSNR_V := 10 * math.Log10(float64(255*255)/MSE_V)
		PSNR := (4*PSNR_Y + PSNR_U + PSNR_V) / 6.0

		fmt.Printf("Frame %03d: PSNR_Y:%f, PSNR_U:%f, PSNR_V:%f, PSNR:%f\n", i, PSNR_Y, PSNR_U, PSNR_V, PSNR)

		TOTAL_PSNR_Y += PSNR_Y
		TOTAL_PSNR_U += PSNR_U
		TOTAL_PSNR_V += PSNR_V
		TOTAL_PSNR += PSNR
	}

	TOTAL_PSNR_Y /= float64(framenum)
	TOTAL_PSNR_U /= float64(framenum)
	TOTAL_PSNR_V /= float64(framenum)
	TOTAL_PSNR /= float64(framenum)

	fmt.Printf("===============================================================================\n")
	fmt.Printf("Total %03d: PSNR_Y:%f, PSNR_U:%f, PSNR_V:%f, PSNR:%f\n", framenum, TOTAL_PSNR_Y, TOTAL_PSNR_U, TOTAL_PSNR_V, TOTAL_PSNR)
}
