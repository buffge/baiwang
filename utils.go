package baiwang

func trimRightBytes(bts []byte) []byte {
	length := len(bts)
	for i := length; i > 0; i-- {
		if bts[i-1] == 0 {
			bts = bts[0 : i-1]
			continue
		}
		break
	}
	return bts
}
