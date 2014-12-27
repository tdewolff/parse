var buf []byte
var pos int
var end int
var err error
var readErr error

func Read(i int) byte {
	if pos + i >= len(buf) {
		if readErr != nil {
			err = readErr
			return 0
		}

		// reallocate a new buffer (possibly larger)
		c := cap(buf)
		d := i
		var buf1 []byte
		if 2*d > c {
			if 2*c > maxBuf {
				err = ErrBufferExceeded
				return 0
			}
			buf1 = make([]byte, d, 2*c)
		} else {
			buf1 = buf[:d]
		}
		copy(buf1, buf[pos:pos+i])

		// Read in to fill the buffer till capacity
		var n int
		n, readErr = r.Read(buf1[d:cap(buf1)])
		pos, buf = 0, buf1[:d+n]
	}
	return buf[pos+i]
}

func readRune() {
	b := Read(end)
  	if b < 0xC0 {
		end += 1
	} else if b < 0xE0 {
		end += 2
	} else if b < 0xF0 {
		end += 3
	} else {
		end += 4
	}
}

func readComment() bool {
	if Read(end) != '/' || Read(end+1) != '*' {
		return false
	}
	end += 2
	
	for {
		if Peek(end) == '*' && Peek(end+1) == '/' {
			end += 2
			break
		} else if err != nil {
			break
		}
		end++
	}
	return true
}

func readCommentOrRune() {
	if !readComment() {
		readRune()
	}
}
