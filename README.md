# &pi;conic

Generate project icon from image.

## Installation

Require go 1.22+

```shell
go install github.com/mawngo/piconic@latest
```

## Usage

Generate icon using image

```shell
> piconic .\my-image.jpeg
```

Or generate for all images in directory

```shell
> piconic .\my-dir
```

## Options

```
> piconic --help
Generate icon from images

Usage:
  piconic [files...] [flags]

Flags:
  -b, --bg string        Background color [transparent, hex, material color name like Yellow500 or svg 1.1 color name like yellow] (default "#f1f5f9")
      --debug            Enable debug mode
  -h, --help             help for piconic
  -o, --out string       Output directory name (default ".")
  -w, --overwrite        Overwrite output if exists
  -p, --padding uint     Padding of the icon image (by % of the size) (default 10)
      --padx int         Additional padding to the x axis (by % of the size)
      --pady int         Additional padding to the y axis (by % of the size)
  -r, --round uint       Round the output image (by % of the size)
  -s, --size uint        Size of the output image (default 200)
      --src-round uint   Round the source image (by % of the size)
      --trim string      List of color to trim when process image (default "transparent")
```

## Examples

### Generate simple icon:

```
piconic eyes.png
```

```shell
5:31PM INF Processing img=eyes.png dimension=160x160 bg=#f1f5f9 size=200
5:31PM INF Processing completed took=5.339563ms
```

| Original              | Icon                                  |
|-----------------------|---------------------------------------|
| ![eyes.png](eyes.png) | ![eyes.200pc10.png](eyes.200pc10.png) |

### Customized generation:

```
piconic cat.jpg --round=20 --src-round=100 --bg=Orange500 --padding=20 --size=480
```

```shell
5:32PM INF Processing img=cat.jpg dimension=481x480 bg=Orange500 size=480
5:32PM INF Processing completed took=29.352891ms
```

| Original            | Icon                                |
|---------------------|-------------------------------------|
| ![cat.jpg](cat.jpg) | ![cat.480pc20.png](cat.480pc20.png) |
