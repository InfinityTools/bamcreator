BAM filters
~~~~~~~~~~~

Introduction
~~~~~~~~~~~~

BAM Creator supports a good number of filters that can be used to alter pixel content and/or frame properties, such
as dimension or center information.

Each option is identified by a unique name and a variable number of options. Filter and option names are not case-
sensitive.

Boolean values are represented by the constants "true" and "false" or numeric values where 0 indicates "false" and
any non-zero value indicates "true".

Numeric values can be specified in decimal notation, hexadecimal notation (with 0x prefix) or octal notation (using
0 prefix).

Floating point values can be specified with or without fractional part and with or without exponent.

Color definitions can be defined as a composite ARGB value, for ease of use in hexadecimal notation 0xAARRGGBB, or as
a sequence of color values in order b, g, r, a, each in range [0, 255]. 'a' is optional and assumed to be fully opaque
when skipped.


Filter description
~~~~~~~~~~~~~~~~~~

1. Brightness
Description: This filter adjusts brightness of frame content.
Filter name: brightness
Filter options:
Level:  A numeric value in range [-255, 255]. Negative values will decrease brightness, positive values will increase
        brightness. Default: 0


2. Contrast
Description: This filter adjusts contrast of frame content.
Filter name: contrast
Filter options:
Level:  A numeric value in range [-255, 255]. Negative values will decrease contrast, positive values will increase
        contrast. Default: 0


3. Gamma
Description: This filter adjusts gamma of frame content.
Filter name: gamma
Filter options:
Level:  A floating point number in range [0.0001, 5.0]. Values below 1.0 will decrease intensity, values above 1.0 will
        increase intensity. Default: 1.0


4. Hue
Description: This filter changes the hue, i.e. shade of the colors for each pixel, of frame content.
Filter name: hue
Filter options:
Level:  A numeric value in range [-180, 180]. Range of hue is continuous in the sense that a value of 180 is equal to
        -180. Default: 0


5. Saturation
Description: This filter adjusts saturation, i.e. color strength, of frame content.
Filter name: saturation
Filter options:
Level:  A numeric value in range [-100, 100]. Negative values will decrease saturation, positive values will increase
        saturation. Default: 0


6. Lightness
Description: This filter adjusts lightness, i.e. the luminance aspect of colors, of frame content.
Filter name: lightness
Filter options:
Level:  A numeric value in range [-100, 100]. Negative values will decrease lightness, positive values will increase
        lightness. Default: 0


7. Balance
Description: This filter adjusts intensity of individual color omponents. For example, reducing the red intensity
             causes the green and blue portion of the color to become more dominant.
Filter name: balance
Filter options:
Red:    A numeric value in range [-255, 255]. Negative values will decrease red intensity. Positive values will
        increase red intensity. Default: 0
Green:  A numeric value in range [-255, 255]. Negative values will decrease green intensity. Positive values will
        increase green intensity. Default: 0
Blue:   A numeric value in range [-255, 255]. Negative values will decrease blue intensity. Positive values will
        increase blue intensity. Default: 0


8. Invert colors
Description: This filter inverts colors of frame content. You can control inversion for each color component.
Filter name: invert
Filter options:
Red:    A boolean value. Indicates whether to invert red intensity of a pixel. Default: true
Green:  A boolean value. Indicates whether to invert green intensity of a pixel. Default: true
Blue:   A boolean value. Indicates whether to invert blue intensity of a pixel. Default: true
Alpha:  A boolean value. Indicates whether to invert alpha intensity of a pixel. Default: false


9. Replace colors
Description: A simple "search-and-replace" filter. It replaces all instances of "Match" by "Color" for each frame.
Filter name: replace
Filter options:
Match:  Specifies the color value to be replaced. Default: 0xff00ff00 (green)
Color:  Specifies the color value that should replace "match". Default: 0 (transparent)


10. Posterize
Description: This filter discards the least significant bits of a color value to create a posterization effect.
Filter name: posterize
Filter options:
Level:  A numeric value in range [0, 7] that defines posterization strength. Higher levels result in a stronger
        posterization effect. Default: 0


11. Alpha to color conversion
Description: This filter simulates translucency effects without involving an alpha channel. It is mainly intended for
             use in visual effects (e.g. applied via VVC), and is compatible with the classic IE games.
             In technical terms, this filter premultiplies RGB values of the source frame with alpha and discards
             alpha afterwards. Full transparency (alpha=0) will be preserved.
             Note: The filter does nothing if the input graphics does not contain translucent pixels.
Filter name: alpha2color
Filter options: n/a


12. Color to alpha conversion
Description: This filter calculates alpha from the color value. It is basically the counterpart of the "alpha2color"
             filter. Only fully opaque color values (alpha=255) will be considered.
Filter name: color2alpha
Filter options: n/a


13. Canvas
Description: This filter allows you to modify the canvas, i.e. the unused region around frame content.
Filter name: canvas
Filter options:
Trim:             A boolean value that indicates whether to remove transparent pixels around BAM frames. Default: false
Borderleft:       A positive numeric value. Specify to add a fixed amount of pixels to the left side of BAM frames.
                  Default: 0
Bordertop:        A positive numeric value. Specify to add a fixed amount of pixels to the top side of BAM frames.
                  Default: 0
Borderright:      A positive numeric value. Specify to add a fixed amount of pixels to the right side of BAM frames.
                  Default: 0
Borderbottom:     A positive numeric value. Specify to add a fixed amount of pixels to the bottom side of BAM frames.
                  Default: 0
Minwidth:         A positive numeric value. All BAM frames will be at least this amount wide. Default: 0
Minheight:        A positive numeric value. All BAM frames will be at least this amount high. Default: 0
Horizontalalign:  Specifies how frame content should be aligned if "MinWidth" increases width of a BAM frame. Valid
                  constants are "left", "center" and "right". Default: center
Verticalalign:    Specifies how frame content should be aligned if "MinHeight" increases height of a BAM frame. Valid
                  constants are "top", "center" and "bottom". Default: center
UpdateCenter:     A boolean value that indicates whether frame center position should be adjusted to match frame
                  content. Default: true

14. Mirror
Description: This filter mirrors frame content either horizontally, vertically, or on both axis.
Filter name: mirror
Filter options:
Horizontal:   A boolean value. Indicates whether to mirror frames horizontally. Default: false
Vertical:     A boolean value. Indicates whether to mirror frames vertically. Default: false
UpdateCenter: A boolean value that indicates whether frame center position should be adjusted to match frame content.
              Default: true


15. Rotate
Description: This filter rotates frame content around the center. Orthogonal angles (90/180/270) are always applied
             without loss in quality. Non-orthogonal operation will result in some quality degradation.
Filter name: rotate
Filter options:
Angle:        A floating point number where range 0 to 360 defines a a full circle. Default: 0
Interpolate:  A boolean value that indicates whether to interpolate rotated pixels. Default: false
Background:   A color value. It specifies the background color for regions that are not covered by source frame pixels.
              Default: 0 (transparent)
UpdateCenter: A boolean value that indicates whether frame center position should be adjusted to match frame content.
              Default: true


16. Resize
Description: Changes dimension of frames and their content by a specified scaling factor. There are several scaling
             algorithms available.
Filter name: resize
Filter options:
Type:         Specifies the scaling algorithm to use. Default: nearest
              Supported types are:
              Nearest:  Nearest neighbor scaling. This simple and fast algorithm duplicates or skips pixels as a whole
                        depending on scaling factor.
              Bilinear: Bilinear filtering. Interpolates neighboring pixel values to produces a smooth image.
              Bicubic:  Applies more advanced calculations on a wider range of neighboring pixels to produce more
                        precise results thant bilinear filtering.
              ScaleX:   A pixel-art filter that works only on integer scaling factors that are a multiple of 2 or 3.
                        Both width and height factors must be identical. It works best for drawn images with clear
                        structures.
ScaleWidth:   A positive floating point number used for horizontal scaling. A value < 1 reduces frame width, a
              value > 1 expands frame width. Default: 1.0
ScaleHeight:  A positive floating point number used for vertical scaling. A value < 1 reduces frame height, a
              value > 1 expands frame height. Default: 1.0
Background:   An optional color value to be used for transparent region. Only needed for pixel-art filter types.
              Default: 0 (transparent)
UpdateCenter: A boolean value that indicates whether frame center position should be adjusted to match frame content.
              Default: true


17. Translate
Description: This filter moves the center position by the specified amount.
Filter name: translate
Filter options:
X:    A numeric value that specifies the amount of pixels to move the center point in horizontal direction. Negative
      values move towards the left side, positive values move towards the right side. Default: 0
Y:    A numeric value that specifies the amount of pixels to move the center point in vertical direction. Negative
      values move towards the top, positive values move towards the bottom. Default: 0
Note: Moving the center point in one direction will result in the frame content being moved to the opposite direction.


18. Split BAM frames
Description: This filter allows you to split BAM frames into multiple segments. This might be useful for large
             multi-part creature animations or item description images.
             Note: Because of technical reasons you can only output one segment per conversion. To convert the
                   remaining segments you can either adapt filter options "SegmentX" and "SegmentY" manually after
                   each pass, or override options "SegmentX" and "SegmentY" via command line options. See readme.txt
                   for more information.
Filter name: split
Filter options:
SplitW:   A numeric value in range [0, 7]. It defines the number of splits to perform in horizontal direction. Default: 0
SplitH:   A numeric value in range [0, 7]. It defines the number of splits to perform in vertical direction. Default: 0
SegmentX: A numeric value in range [0, SplitW]. It specifies the column of the segment to return. Column 0 is left-most
          column. Default: 0
SegmentY: A numeric value in range [0, SplitH]. It specifies the row of the segment to return. Row 0 is top-most row.
          Default: 0
