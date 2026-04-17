package renderer

import "godocmirrortranslator/internal/domain"

const (
	PortraitWidthMM  = 210.0
	PortraitHeightMM = 297.0
)

func A4Dimensions(orientation domain.Orientation) (float64, float64) {
	if orientation == domain.OrientationLandscape {
		return PortraitHeightMM, PortraitWidthMM
	}
	return PortraitWidthMM, PortraitHeightMM
}

func FitScale(pageWidth, pageHeight, sourceWidth, sourceHeight float64) float64 {
	if sourceWidth <= 0 || sourceHeight <= 0 {
		return 1
	}
	widthScale := pageWidth / sourceWidth
	heightScale := pageHeight / sourceHeight
	if widthScale < heightScale {
		return widthScale
	}
	return heightScale
}
