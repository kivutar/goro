package res

import "strconv"

var grfMask = [8]byte{0x80, 0x40, 0x20, 0x10, 0x08, 0x04, 0x02, 0x01}

func decryptGRFFull(buf []byte, alignedLen, packedLen uint32) {
	nblocks := int(alignedLen) >> 3
	digits := len(strconv.FormatUint(uint64(packedLen), 10))
	var cycle int
	switch {
	case digits < 3:
		cycle = 3
	case digits < 5:
		cycle = digits + 1
	case digits < 7:
		cycle = digits + 9
	default:
		cycle = digits + 15
	}

	for i := 0; i < 20 && i < nblocks; i++ {
		decryptGRFBlock(buf, i*8)
	}
	for i, j := 20, 0; i < nblocks; i++ {
		if i%cycle == 0 {
			decryptGRFBlock(buf, i*8)
			continue
		}
		if j == 7 {
			shuffleGRFBlock(buf, i*8)
			j = 0
		}
		j++
	}
}

func decryptGRFHeader(buf []byte, alignedLen uint32) {
	nblocks := int(alignedLen) >> 3
	for i := 0; i < 20 && i < nblocks; i++ {
		decryptGRFBlock(buf, i*8)
	}
}

func decryptGRFBlock(src []byte, index int) {
	initialGRFPermutation(src, index)
	grfRoundFunction(src, index)
	finalGRFPermutation(src, index)
}

func initialGRFPermutation(src []byte, index int) {
	var tmp [8]byte
	for i := 0; i < 64; i++ {
		j := int(grfInitialPermutationTable[i] - 1)
		if src[index+((j>>3)&7)]&grfMask[j&7] != 0 {
			tmp[(i>>3)&7] |= grfMask[i&7]
		}
	}
	copy(src[index:index+8], tmp[:])
}

func finalGRFPermutation(src []byte, index int) {
	var tmp [8]byte
	for i := 0; i < 64; i++ {
		j := int(grfFinalPermutationTable[i] - 1)
		if src[index+((j>>3)&7)]&grfMask[j&7] != 0 {
			tmp[(i>>3)&7] |= grfMask[i&7]
		}
	}
	copy(src[index:index+8], tmp[:])
}

func grfRoundFunction(src []byte, index int) {
	var tmp [8]byte
	copy(tmp[:], src[index:index+8])
	grfExpansion(tmp[:], 0)
	grfSubstitutionBox(tmp[:], 0)
	grfTransposition(tmp[:], 0)

	src[index+0] ^= tmp[4]
	src[index+1] ^= tmp[5]
	src[index+2] ^= tmp[6]
	src[index+3] ^= tmp[7]
}

func grfExpansion(src []byte, index int) {
	var tmp [8]byte
	tmp[0] = ((src[index+7] << 5) | (src[index+4] >> 3)) & 0x3f
	tmp[1] = ((src[index+4] << 1) | (src[index+5] >> 7)) & 0x3f
	tmp[2] = ((src[index+4] << 5) | (src[index+5] >> 3)) & 0x3f
	tmp[3] = ((src[index+5] << 1) | (src[index+6] >> 7)) & 0x3f
	tmp[4] = ((src[index+5] << 5) | (src[index+6] >> 3)) & 0x3f
	tmp[5] = ((src[index+6] << 1) | (src[index+7] >> 7)) & 0x3f
	tmp[6] = ((src[index+6] << 5) | (src[index+7] >> 3)) & 0x3f
	tmp[7] = ((src[index+7] << 1) | (src[index+4] >> 7)) & 0x3f
	copy(src[index:index+8], tmp[:])
}

func grfSubstitutionBox(src []byte, index int) {
	var tmp [8]byte
	for i := 0; i < 4; i++ {
		tmp[i] = (grfSubstitutionBoxTable[i][src[index+i*2]] & 0xf0) |
			(grfSubstitutionBoxTable[i][src[index+i*2+1]] & 0x0f)
	}
	copy(src[index:index+8], tmp[:])
}

func grfTransposition(src []byte, index int) {
	var tmp [8]byte
	for i := 0; i < 32; i++ {
		j := int(grfTranspositionTable[i] - 1)
		if src[index+(j>>3)]&grfMask[j&7] != 0 {
			tmp[(i>>3)+4] |= grfMask[i&7]
		}
	}
	copy(src[index:index+8], tmp[:])
}

func shuffleGRFBlock(src []byte, index int) {
	var tmp [8]byte
	tmp[0] = src[index+3]
	tmp[1] = src[index+4]
	tmp[2] = src[index+6]
	tmp[3] = src[index+0]
	tmp[4] = src[index+1]
	tmp[5] = src[index+2]
	tmp[6] = src[index+5]
	tmp[7] = grfShuffleTable[src[index+7]]
	copy(src[index:index+8], tmp[:])
}

var grfInitialPermutationTable = [64]byte{
	58, 50, 42, 34, 26, 18, 10, 2, 60, 52, 44, 36, 28, 20, 12, 4,
	62, 54, 46, 38, 30, 22, 14, 6, 64, 56, 48, 40, 32, 24, 16, 8,
	57, 49, 41, 33, 25, 17, 9, 1, 59, 51, 43, 35, 27, 19, 11, 3,
	61, 53, 45, 37, 29, 21, 13, 5, 63, 55, 47, 39, 31, 23, 15, 7,
}

var grfFinalPermutationTable = [64]byte{
	40, 8, 48, 16, 56, 24, 64, 32, 39, 7, 47, 15, 55, 23, 63, 31,
	38, 6, 46, 14, 54, 22, 62, 30, 37, 5, 45, 13, 53, 21, 61, 29,
	36, 4, 44, 12, 52, 20, 60, 28, 35, 3, 43, 11, 51, 19, 59, 27,
	34, 2, 42, 10, 50, 18, 58, 26, 33, 1, 41, 9, 49, 17, 57, 25,
}

var grfTranspositionTable = [32]byte{
	16, 7, 20, 21, 29, 12, 28, 17, 1, 15, 23, 26, 5, 18, 31, 10,
	2, 8, 24, 14, 32, 27, 3, 9, 19, 13, 30, 6, 22, 11, 4, 25,
}

var grfSubstitutionBoxTable = [4][64]byte{
	{
		0xef, 0x03, 0x41, 0xfd, 0xd8, 0x74, 0x1e, 0x47, 0x26, 0xef, 0xfb, 0x22, 0xb3, 0xd8, 0x84, 0x1e,
		0x39, 0xac, 0xa7, 0x60, 0x62, 0xc1, 0xcd, 0xba, 0x5c, 0x96, 0x90, 0x59, 0x05, 0x3b, 0x7a, 0x85,
		0x40, 0xfd, 0x1e, 0xc8, 0xe7, 0x8a, 0x8b, 0x21, 0xda, 0x43, 0x64, 0x9f, 0x2d, 0x14, 0xb1, 0x72,
		0xf5, 0x5b, 0xc8, 0xb6, 0x9c, 0x37, 0x76, 0xec, 0x39, 0xa0, 0xa3, 0x05, 0x52, 0x6e, 0x0f, 0xd9,
	},
	{
		0xa7, 0xdd, 0x0d, 0x78, 0x9e, 0x0b, 0xe3, 0x95, 0x60, 0x36, 0x36, 0x4f, 0xf9, 0x60, 0x5a, 0xa3,
		0x11, 0x24, 0xd2, 0x87, 0xc8, 0x52, 0x75, 0xec, 0xbb, 0xc1, 0x4c, 0xba, 0x24, 0xfe, 0x8f, 0x19,
		0xda, 0x13, 0x66, 0xaf, 0x49, 0xd0, 0x90, 0x06, 0x8c, 0x6a, 0xfb, 0x91, 0x37, 0x8d, 0x0d, 0x78,
		0xbf, 0x49, 0x11, 0xf4, 0x23, 0xe5, 0xce, 0x3b, 0x55, 0xbc, 0xa2, 0x57, 0xe8, 0x22, 0x74, 0xce,
	},
	{
		0x2c, 0xea, 0xc1, 0xbf, 0x4a, 0x24, 0x1f, 0xc2, 0x79, 0x47, 0xa2, 0x7c, 0xb6, 0xd9, 0x68, 0x15,
		0x80, 0x56, 0x5d, 0x01, 0x33, 0xfd, 0xf4, 0xae, 0xde, 0x30, 0x07, 0x9b, 0xe5, 0x83, 0x9b, 0x68,
		0x49, 0xb4, 0x2e, 0x83, 0x1f, 0xc2, 0xb5, 0x7c, 0xa2, 0x19, 0xd8, 0xe5, 0x7c, 0x2f, 0x83, 0xda,
		0xf7, 0x6b, 0x90, 0xfe, 0xc4, 0x01, 0x5a, 0x97, 0x61, 0xa6, 0x3d, 0x40, 0x0b, 0x58, 0xe6, 0x3d,
	},
	{
		0x4d, 0xd1, 0xb2, 0x0f, 0x28, 0xbd, 0xe4, 0x78, 0xf6, 0x4a, 0x0f, 0x93, 0x8b, 0x17, 0xd1, 0xa4,
		0x3a, 0xec, 0xc9, 0x35, 0x93, 0x56, 0x7e, 0xcb, 0x55, 0x20, 0xa0, 0xfe, 0x6c, 0x89, 0x17, 0x62,
		0x17, 0x62, 0x4b, 0xb1, 0xb4, 0xde, 0xd1, 0x87, 0xc9, 0x14, 0x3c, 0x4a, 0x7e, 0xa8, 0xe2, 0x7d,
		0xa0, 0x9f, 0xf6, 0x5c, 0x6a, 0x09, 0x8d, 0xf0, 0x0f, 0xe3, 0x53, 0x25, 0x95, 0x36, 0x28, 0xcb,
	},
}

var grfShuffleTable = func() [256]byte {
	var out [256]byte
	for i := range out {
		out[i] = byte(i)
	}
	swaps := [][2]byte{
		{0x00, 0x2b},
		{0x6c, 0x80},
		{0x01, 0x68},
		{0x48, 0x77},
		{0x60, 0xff},
		{0xb9, 0xc0},
		{0xfe, 0xeb},
	}
	for _, swap := range swaps {
		out[swap[0]] = swap[1]
		out[swap[1]] = swap[0]
	}
	return out
}()
