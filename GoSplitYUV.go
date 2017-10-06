package main

import (
	"fmt"
	"os"
	"strconv"
)

func main() {
	if len(os.Args) < 4 {
		print("Usage: GoSplitYUV.exe input.yuv output.yuv skipbytes")
		return
	}

	skipbytes, err := strconv.Atoi(os.Args[3])
	if err != nil {
		print("skipbytes is not digital")
		return
	}

	var inputFile, outputFile *os.File

	inputFile, err = os.Open(os.Args[1])
	if err != nil {
		fmt.Printf("\nFailed to open input file %s\n", os.Args[1])
		return
	}
	defer inputFile.Close()

	outputFile, err = os.Create(os.Args[2])
	if err != nil {
		fmt.Printf("\nFailed to create output file %s\n", os.Args[2])
		return
	}
	defer outputFile.Close()

	var read_buffer_size int

	_, err = inputFile.Seek(int64(skipbytes), 0)

	// if read_buffer_size != int64(skipbytes) {
	// 	print("Failed to read yuv!")
	// 	return
	// }

	buffer := make([]byte, 1024*1024)
	read_buffer_size = 1
	for read_buffer_size > 0 {
		read_buffer_size, err = inputFile.Read(buffer)
		if read_buffer_size > 0 {
			outputFile.Write(buffer[0:read_buffer_size])
		}
	}

}
