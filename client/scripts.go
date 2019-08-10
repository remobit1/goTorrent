package client

// ConvertIntToUint64 converts an int to an unsigned int64
func convertIntToUint64(i int) uint64 {
	rep64Int := int64(i)
	return uint64(rep64Int)
}

// ConvertIntToUint32 converts an int to an unsigned int32
func convertIntToUint32(i int) uint32 {
	rep32Int := int32(i)
	return uint32(rep32Int)
}

// ConvertIntToUint16 converts an int to an unsigned int16
func convertIntToUint16(i int) uint16 {
	rep16Int := int16(i)
	return uint16(rep16Int)
}

// turns a slice of bytes into an int slice representation of inividual bits. Useful for bitfield evaluation.
func getBitsFromByteSlice(bs []byte) []int {
	r := make([]int, len(bs)*8)
	for i, b := range bs {
		for j := 0; j < 8; j++ {
			r[i*8+j] = int(b >> uint(7-j) & 0x01)
		}
	}
	return r
}
