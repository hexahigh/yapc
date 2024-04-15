package hash

import (
	"image"

	"github.com/disintegration/imaging"
)

// Ahash calculates the average hash of an image. The image is first grayscaled,
// then scaled down to "hashLen" for the width and height. Then, the average value
// of the pixels is computed, and if a pixel is above the average, a 1 is appended
// to the byte array; a 0 otherwise.
func Ahash(img image.Image, hashLen int) ([]byte, error) {
	var sum uint32                        // Sum of the pixels
	numbits := hashLen * hashLen          // Perform the hashLen^2 operation once
	bitArray, err := NewBitArray(numbits) // Resultant byte array init
	if err != nil {
		return nil, err
	}

	// As the average is being computed, create & populate an array
	// of pixels to optimize runtime
	var pixelArray []uint32

	// Grayscale and resize
	res := imaging.Grayscale(img)
	res = imaging.Resize(res, hashLen, hashLen, imaging.Lanczos)

	// Iterate over every pixel to generate the sum.
	// Additionally, store every pixel into an array for faster re-computation
	for x := 0; x < hashLen; x++ {
		for y := 0; y < hashLen; y++ {
			r, _, _, _ := res.At(x, y).RGBA()  // r = g = b since the image is grayscaled
			sum += r                           // increment the sum
			pixelArray = append(pixelArray, r) // append the pixel
		}
	}

	// Compute the average
	avg := sum / uint32(numbits)

	// For every pixel, check if it's below or above the average
	for _, pix := range pixelArray {
		if pix > avg {
			bitArray.AppendBit(1) // If above, append 1
		} else {
			bitArray.AppendBit(0) // else append 0
		}
	}

	return bitArray.GetArray(), nil
}

func Dhash(img image.Image, hashLen int) ([]byte, error) {
	imgGray := imaging.Grayscale(img) // Grayscale image first for performance

	// Calculate both horizontal and vertical gradients
	horiz, err1 := horizontalGradient(imgGray, hashLen)
	vert, err2 := verticalGradient(imgGray, hashLen)

	if err1 != nil {
		return nil, err1
	}
	if err2 != nil {
		return nil, err2
	}

	// Return the concatenated horizontal and vertical hash
	return append(horiz, vert...), nil
}

// DhashHorizontal returns the result of a horizontal gradient hash.
// 'img' is an Image object returned by opening an image file using OpenImg().
// 'hashLen' is the size that the image will be shrunk to. It must be a non-zero multiple of 8.
func DhashHorizontal(img image.Image, hashLen int) ([]byte, error) {
	imgGray := imaging.Grayscale(img)           // Grayscale image first
	return horizontalGradient(imgGray, hashLen) // horizontal diff gradient
}

// DhashVertical returns the result of a vertical gradient hash.
// 'img' is an Image object returned by opening an image file using OpenImg().
// 'hashLen' is the size that the image will be shrunk to. It must be a non-zero multiple of 8.
func DhashVertical(img image.Image, hashLen int) ([]byte, error) {
	imgGray := imaging.Grayscale(img)         // Grayscale image first
	return verticalGradient(imgGray, hashLen) // vertical diff gradient
}

// horizontalGradient performs a horizontal gradient diff on a grayscaled image
func horizontalGradient(img image.Image, hashLen int) ([]byte, error) {
	// Width and height of the scaled-down image
	width, height := hashLen+1, hashLen

	// Downscale the image by 'hashLen' amount for a horizonal diff.
	res := imaging.Resize(img, width, height, imaging.Lanczos)

	// Create a new bitArray
	bitArray, err := NewBitArray(hashLen * hashLen)
	if err != nil {
		return nil, err
	}

	var prev uint32 // Variable to store the previous pixel value

	// Calculate the horizonal gradient difference
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			// Since the image is grayscaled, r = g = b
			r, _, _, _ := res.At(x, y).RGBA() // Get the pixel at (x,y)

			// If this is not the first value of the current row, then
			// compare the gradient difference from the previous one
			if x > 0 {
				if prev < r {
					bitArray.AppendBit(1) // if it's smaller, append '1'
				} else {
					bitArray.AppendBit(0) // else append '0'
				}
			}
			prev = r // Set this current pixel value as the previous one
		}
	}
	return bitArray.GetArray(), nil
}

// verticalGradient performs a vertical gradient diff on a grayscaled image
func verticalGradient(img image.Image, hashLen int) ([]byte, error) {
	// Width and height of the scaled-down image
	width, height := hashLen, hashLen+1

	// Downscale the image by 'hashLen' amount for a vertical diff.
	res := imaging.Resize(img, width, height, imaging.Lanczos)

	// Create a new bitArray
	bitArray, err := NewBitArray(hashLen * hashLen)
	if err != nil {
		return nil, err
	}

	var prev uint32 // Variable to store the previous pixel value

	// Calculate the vertical gradient difference
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			// Since the image is grayscaled, r = g = b
			r, _, _, _ := res.At(x, y).RGBA() // Get the pixel at (x,y)

			// If this is not the first value of the current column, then
			// compare the gradient difference from the previous one
			if y > 0 {
				if prev < r {
					bitArray.AppendBit(1) // if it's smaller, append '1'
				} else {
					bitArray.AppendBit(0) // else append '0'
				}
			}
			prev = r // Set this current pixel value as the previous one
		}
	}
	return bitArray.GetArray(), nil
}
