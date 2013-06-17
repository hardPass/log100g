package main

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"runtime"
	"strconv"
	// "sync"
	"time"
)

const (
	SIZE_PER_BUF = 1 << 20
)

var (
	toDisk     = make(chan []byte, 10)
	waitToDisk = make(chan bool)
	parts      = make([]*os.File, 256, 256)
	bufs       = make([]*bytes.Buffer, 256, 256)

	round_w  = 1 << 20 * 10 // change rand  by seed every round
	rounds_w = 0

	round_r  = 1 << 29 // change rand  by seed every round
	rounds_r = 0

	start int64 // start time
)

func resolveIp4(ip []string) (int64, []byte, error) {
	var err error
	var n int64
	b := make([]byte, 3)
	n, err = strconv.ParseInt(ip[1], 10, 0)
	if err != nil {
		return 0, nil, err
	}
	b[0] = byte(n)

	n, err = strconv.ParseInt(ip[2], 10, 0)
	if err != nil {
		return 0, nil, err
	}
	b[1] = byte(n)

	n, err = strconv.ParseInt(ip[3], 10, 0)
	if err != nil {
		return 0, nil, err
	}
	b[2] = byte(n)

	first, err := strconv.ParseInt(ip[0], 10, 0)
	if err != nil {
		return 0, nil, err
	}

	return first, b, nil
}

// return len of read,ip0,ip123, not EOF
func resolveLine(r *bufio.Reader) (int, int64, []byte, bool) {
	n := 0
	ip := make([]string, 4, 4)
	for i := 0; i < 3; i++ {
		b, err := r.ReadBytes('.')
		n += len(b)
		if err != nil {
			return n, -1, nil, false
		}

		if len(b) > 4 || len(b) < 2 {
			b, err = r.ReadBytes('\n')
			n += len(b)
			if err != nil {
				return n, -1, nil, false
			}

			return n, -1, nil, true
		}
		ip[i] = string(b[:len(b)-1])
	}

	//	for ip3
	b, err := r.ReadBytes(' ')
	n += len(b)
	if err != nil {
		return n, -1, nil, false
	}

	if len(b) > 4 || len(b) < 2 {
		b, err = r.ReadBytes('\n')
		n += len(b)
		if err != nil {
			return n, -1, nil, false
		}

		return n, -1, nil, true
	}
	ip[3] = string(b[:len(b)-1])

	ip0, ip123, err := resolveIp4(ip)
	if err != nil {
		b, err = r.ReadBytes('\n')
		n += len(b)
		if err != nil {
			return n, -1, nil, false
		}

		return n, -1, nil, true
	}

	b, err = r.ReadBytes('\n')
	n += len(b)
	if err != nil {
		return n, ip0, ip123, false
	}

	return n, ip0, ip123, true
}

func resolveToInt(ip0 int64, b []byte) uint32 {
	return uint32(ip0)<<24 | uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
}

// create part file if necessary
func PartFile(ip0 int) *os.File {
	fi := parts[ip0]
	if fi == nil {
		fi, err := os.OpenFile("./ip_parts/"+strconv.FormatInt(int64(ip0), 10)+".part", os.O_RDWR|os.O_APPEND|os.O_CREATE, 0600)
		// actually , the filename is not very important , any random char sequences could be ok here.
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			return nil
		}
		//defer fi.Close()
		fi.Seek(0, 0)
		parts[ip0] = fi

		return fi
	}

	return fi
}

// close all opened partFile
func closeAllParts() {
	for i := 0; i < len(parts); i++ {
		fi := parts[i]
		if fi != nil {
			go fi.Close()
		}
	}
}

func LoopToDisk() {
	total := 0
	for {
		select {
		case b, _ := <-toDisk:
			if len(b) == 0 {
				waitToDisk <- true
				fmt.Printf("ipToDisk %d MB , all spend: %d ms \n", total/(1<<20), (time.Now().UnixNano()-start)/(1000*1000))
				return
			}
			fi := PartFile(int(b[0]))

			n, err := fi.Write(b[1:])
			if err != nil {
				fmt.Printf("Error: %s\n", err)
				fmt.Printf("Error: %s\n", err.Error())
			}
			total += n
			if total > (rounds_w * round_w) {
				rounds_w++
				fmt.Printf("ipToDisk %d MB , all spend: %d ms \n", total/(1<<20), (time.Now().UnixNano()-start)/(1000*1000))
			}
		}
	}

}

func LoopRead(r *bufio.Reader) {
	total := 0
	for {
		n, ip0, b, notEof := resolveLine(r)

		if ip0 > 0 {
			bufData(int(ip0), b)

			// print ip to cmd or shell console
			// buf := bytes.NewBuffer(make([]byte, 0, 30))
			// buf.WriteString(strconv.FormatInt(ip0, 10))
			// buf.WriteByte('.')
			// buf.WriteString(strconv.FormatInt(int64(b[0]), 10))
			// buf.WriteByte('.')
			// buf.WriteString(strconv.FormatInt(int64(b[1]), 10))
			// buf.WriteByte('.')
			// buf.WriteString(strconv.FormatInt(int64(b[2]), 10))
			// fmt.Println(buf.String())
		}

		total += n
		if total > (rounds_r * round_r) {
			rounds_r++
			fmt.Printf("read from big log %d MB , all spend: %d ms \n", total/(1<<20), (time.Now().UnixNano()-start)/(1000*1000))
		}

		if !notEof {
			break
		}
	}

	// flush all remains in buffers to disk
	for i := 0; i < len(bufs); i++ {
		buf := bufs[i]
		if buf != nil {
			bufs[i] = nil
			toDisk <- buf.Bytes()
		}

	}

	// send a finished signal
	toDisk <- make([]byte, 0, 0)
}
func bufData(ip0 int, b []byte) {
	buf := bufs[ip0]
	if buf == nil {
		buf = bytes.NewBuffer(make([]byte, 0, SIZE_PER_BUF+6))
		buf.WriteByte(byte(ip0)) // first byte is a trick
		bufs[ip0] = buf
	}

	buf.Write(b)
	if buf.Len() >= SIZE_PER_BUF {
		bufs[ip0] = nil
		toDisk <- buf.Bytes()
	}

}

// Convert Ip3 []byte to uint32
func Ip3BytesToUint32(b []byte) uint32 {
	return uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2])
}
func Ip4Str(v uint32) string {
	b := make([]byte, 4)
	b[0] = byte(v >> 24)
	b[1] = byte(v >> 16)
	b[2] = byte(v >> 8)
	b[3] = byte(v)

	return strconv.FormatInt(int64(b[0]), 10) + "." +
		strconv.FormatInt(int64(b[1]), 10) + "." +
		strconv.FormatInt(int64(b[2]), 10) + "." +
		strconv.FormatInt(int64(b[3]), 10)
}

func maxIpInPart(part *os.File) (ip3 int, max int) {
	ip3Array := make([]int, 1<<24, 1<<24)
	b := make([]byte, 1<<16*3)
	part.Seek(0, 0)
	for {
		n, err := part.Read(b)
		if n > 0 {
			for i := 0; i < n; i += 3 {
				ip3Array[int(Ip3BytesToUint32(b[i:i+3]))]++
			}

		}
		if err != nil {
			break
		}
	}

	for i := 0; i < len(ip3Array); i++ {
		if ip3Array[i] > max {
			max = ip3Array[i]
			ip3 = i
		}
	}
	return
}

func main() {
	if err := os.Mkdir("./ip_parts", 0777); err != nil {
		fmt.Printf("Error: %v\n", err)
	}
	fmt.Printf("./ip_parts/ has been created!\n")

	fi, err := os.Open("./100g.log")
	if err != nil {
		fmt.Printf("Error: %s\n", err)
		return
	}
	defer fi.Close()
	fi.Seek(0, 0)

	r := bufio.NewReader(fi)

	runtime.GOMAXPROCS(3)

	fmt.Println("Begin...")
	start = time.Now().UnixNano()

	go LoopToDisk()

	LoopRead(r)
	defer closeAllParts()

	<-waitToDisk

	ip4List := make([]uint32, 0)
	max := 0
	for i := 0; i < len(parts); i++ {
		part := parts[i]
		if part == nil {
			continue
		}

		// fmt.Printf("process part %d, all spend: %d\n", i, (time.Now().UnixNano()-start)/(1000*1000))

		ip3, m := maxIpInPart(part)
		if m > max {
			max = m
			ip4List = make([]uint32, 1)
			ip4List[0] = uint32(i)<<24 | uint32(ip3)
		} else if m == max {
			ip4List = append(ip4List, uint32(i)<<24|uint32(ip3))
		}
	}

	end := time.Now().UnixNano()
	fmt.Printf("Done in %d ms!\n", (end-start)/(1000*1000))
	fmt.Printf("Max count  is: %d \n", max)
	fmt.Printf("IP list is :\n")
	for i := 0; i < len(ip4List); i++ {
		fmt.Printf(" %s \n", Ip4Str(ip4List[i]))
	}

}
