# BAM Creator
*A command line tool for creating BAM V1 or BAM V2 files.*

# About

*BAM Creator* is a command line tool that allows you to generate BAM files, based on settings defined in configuration files. It features:
- BAM file output in BAM V1 or BAM V2 format.
- Supported input file formats: BAM, BMP, (animated) GIF, JPG and PNG.
- Full control over the resulting BAM structure, such as individual frame, frame center and frame cycle definitions.
- BAM V1: Palette generation with or without alpha support.
- BAM V2: PVRZ texture generation.
- Various quality and compression settings for both BAM output types.
- High performance by taking advantage of multithreading.
- A powerful filter processor: Choose one or more filters out of over 15 different filter types to improve, alter or optimize the resulting BAM output.

Settings are parsed from configuration data in XML or JSON format. Configuration can be read from files or fetched directly from standard input. A fully commented `example.xml` and the json counterpart are included in the `examples` folder.

A great number of configuration settings can be overridden by command line options. Invoke `bamcreator --help` or see `readme.txt` for more information. A detailed description of BAM filters can be found in `readme_filters.txt`.

# Building

The tool requires the [Go compiler](https://golang.org/) to be installed. Code dependencies are automatically resolved when using the included build scripts. However, the packages [go-imagequant](https://github.com/InfinityTools/go-imagequant) and [go-squish](https://github.com/InfinityTools/go-squish) have to be prepared manually before they can be used in the build process.

Install the upx packer (https://github.com/upx/upx/releases) if you want to make use of the `--compression` option of the build scripts.

Use the provided build scripts (`build.cmd` for Windows, `build.sh` for Linux and macOS) to build the binaries. You can specify the build target and a number of options to influence the build process.

Syntax for *build.sh* (the same syntax applies to *build.cmd*):
```
Usage: build.sh [options] [target]

Options:
  --debug         Don't strip debugging symbols
  --nodeps        Don't check dependencies
  --update        Force updating dependencies
  --compress      Compress the binary with upx if available
  --help          This help

Available targets:
  bamcreator (default)
```
The resulting binary will be placed into the folder `./bin/$GOOS/$GOARCH`.

Alternatively you can build binaries directly. Go into the subdirectory `bamcreator` and call `go build`.

## License

*BAM Creator* is released under the BSD 2-clause license. See LICENSE for more details.
