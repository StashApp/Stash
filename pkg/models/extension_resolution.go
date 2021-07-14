package models

type ResolutionRange struct {
	min, max int
}

var resolutionRanges = map[ResolutionEnum]ResolutionRange{
	ResolutionEnum("VERY_LOW"):    {144, 239},
	ResolutionEnum("LOW"):         {240, 359},
	ResolutionEnum("R360P"):       {360, 479},
	ResolutionEnum("STANDARD"):    {480, 539},
	ResolutionEnum("WEB_HD"):      {540, 719},
	ResolutionEnum("STANDARD_HD"): {720, 1079},
	ResolutionEnum("FULL_HD"):     {1080, 1439},
	ResolutionEnum("QUAD_HD"):     {1440, 1919},
	ResolutionEnum("VR_HD"):       {1920, 2159},
	ResolutionEnum("FOUR_K"):      {2160, 2879},
	ResolutionEnum("FIVE_K"):      {2880, 3383},
	ResolutionEnum("SIX_K"):       {3384, 4319},
	ResolutionEnum("EIGHT_K"):     {4320, 8639},
}

// GetMaxResolution returns the maximum width or height that media must be
// to qualify as this resolution.
func (r *ResolutionEnum) GetMaxResolution() int {
	return resolutionRanges[*r].max
}

// GetMinResolution returns the minimum width or height that media must be
// to qualify as this resolution.
func (r *ResolutionEnum) GetMinResolution() int {
	return resolutionRanges[*r].min
}
