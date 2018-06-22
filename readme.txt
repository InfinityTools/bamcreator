BAM Creator
~~~~~~~~~~~

Version:    0.2
Author:     Argent77

Project:    https://github.com/InfinityTools/bamcreator
Download:   https://github.com/InfinityTools/bamcreator/releases


About
~~~~~

BAM Creator is a command line tool that allows you to generate BAM files, based on settings defined in configuration
files. It features:
- BAM file output in BAM V1 or BAM V2 format.
- Supported input file formats: BAM, BMP, (animated) GIF, JPG and PNG.
- Full control over the resulting BAM structure, such as individual frame, frame center and frame cycle definitions.
- BAM V1: Palette generation with or without alpha channel support.
- BAM V2: PVRZ texture generation.
- A great number of settings to fine-tune the resulting BAM file.
- High performance by taking advantage of multithreading.
- A powerful filter processor: Choose one or more filters out of over 15 different filter types to improve, alter or
                               optimize the resulting BAM output.
- A supplementary tool "bamconv" that helps generate configuration data from BAM source files (see readme_bamconv.txt).

Settings are parsed from configuration data in XML or JSON format. Configuration can be read from files or fetched
directly from standard input. A fully commented "example.xml" and the json counterpart are included in the "examples"
folder.

A detailed description of BAM filters can be found in "readme_filters.txt".


Installation
~~~~~~~~~~~~

The mod archive contains binaries for
- Windows:
  - 32-bit: in bin/windows/386
  - 64-bit: in bin/windows/amd64
- Linux:
  - 32-bit: in bin/linux/386
  - 64-bit: in bin/linux/amd64
- macOS:
  - 64-bit: in bin/darwin/amd64

There is no installation required. Simply unpack the binary of your choice into a directory on your system. You can
optionally add the directory to your path to make it easier to work with the tool.


Usage
~~~~~

Usage: bamcreator [options] configfile [configfile2 ...]

Options:
--verbose                   Show additional log messages during the conversion process.
--silent                    Suppress any log messages during the conversion process except for errors.
--log-style                 Print log messages in log style, complete with timestamp and log level.
--threaded                  Enable multithreading for BAM conversion. May speed up the conversion process on multi-core
                            systems. Enabled by default if multiple CPU cores are detected.
--no-threaded               Disable multithreading for BAM conversion.
--optimize level            Optimize BAM output file. Available optimization levels:
                                0     No optimization.
                                1     Remove unreferenced frames.
                                2     Remove duplicate frames.
                                3     Remove similar frames.
                            Optimization levels are cumulative. Default: 0
--bam-version version       Set BAM output version. Can be 1 for BAM V1 or 2 for BAM V2. Overrides setting in the
                            config file.
--bam-output file           Set BAM output file. Overrides setting in the config file.
--bam-pvrz-path path        Set PVRZ output path for BAM V2 output. Overrides setting in the config file.
--bamv1-compress            Enable BAM V1 compression. Overrides setting in the config file.
--bamv1-no-compress         Disable BAM V1 compression. Overrides setting in the config file.
--bamv1-rle type            Set RLE frame compression type. Allowed types:
                               -1     Decide based on resulting frame size
                                0     Always disable RLE encoding
                                1     Always enable RLE encoding
                            Overrides setting in the config file.
--bamv1-alpha               Preserve alpha in BAM V1 palette. Overrides setting in the config file.
--bamv1-no-alpha            Discard alpha in BAM V1 palette. Overrides setting in the config file.
--bamv1-quality-min qmin    Set minimum quality for BAM V1 color quantization. Allowed range: [0, 100]. Overrides
                            setting in the config file.
--bamv1-quality-max qmax    Set maximum quality for BAM V1 color quantization. Allowed range: [0, 100]. Overrides
                            setting in the config file.
--bamv1-speed value         Set speed for palette generation. Allowed range: [1, 10]. Overrides setting in the config
                            file.
--bamv1-dither value        Set dither strength for output graphics. Value must be in range [0.0, 1.0]. Set to 0 to
                            disable. Overrides setting in the config file.
--bamv1-transcolor          Enable to treat the color defined in the config file as transparent. Overrides setting in
                            the config file.
--bamv1-no-transcolor       Don't treat the color defined in the config file as transparent. Overrides setting in the
                            config file.
--bamv1-sort type           Sort palette by the specified type. The following types are recognized: none, lightness,
                            saturation, hue, red, green, blue, alpha. Append _reversed to reverse the sort order.
                            Overrides setting in the config file.
--bamv1-palette file        Specify an external palette. Overrides settings in the config file.
--bamv2-start-index idx     Set start index for PVRZ files generated by BAM V2. Allowed range: [0, 99999]. Overrides
                            setting in the config file.
--bamv2-encoding type       PVRZ pixel encoding type. Available types:
                                0     Determine automatically
                                1     Enforce DXT1 (BC1)
                                2     Enforce DXT3 (BC2) [unsupported by games]
                                3     Enforce DXT5 (BC3)
                            Overrides setting in the config file.
--bamv2-threshold value     Percentage threshold for determining PVRZ pixel encoding type. Allowed range: [0.0, 100.0].
                            Overrides setting in the config file.
--bamv2-quality value       Quality of PVRZ pixel encoding. in range [0, 2], where 0 is lowest quality and 2 is highest
                            quality. Overrides setting in the config file.
--bamv2-weight-alpha        Weight pixels by alpha. May improve visual quality for alpha-blended pixels. Overrides
                            setting in the config file.
--bamv2-no-weight-alpha     Don't weight pixels by alpha. Overrides setting in the config file.
--bamv2-use-metric          Apply perceptual metric to encoded pixels. May improve perceived quality. Overrides setting
                            in the config file.
--bamv2-no-use-metric       Don't apply perceptual metric to encoded pixels. Overrides setting in the config file.
--filter idx:key=value      Set or override a filter option. 'idx' indicates the filter index in the list of filters,
                            starting at index 0. 'key' and 'value' define a single filter option key and value pair.
                            Wrap the whole definition in quotes if it contains spaces. Add multiple --filter instances
                            to set or override multiple filter options.
--help                      Print this help and terminate.
--version                   Print version information and terminate.

Note: Use minus sign (-) in place of configfile to read configuration data from standard input.


Configuration
~~~~~~~~~~~~~

BAM output is primarily controlled by configuration data in XML or JSON format.

Example XML configuration:
<?xml version="1.0" encoding="UTF-8"?>
<generator>
    <output>
        <!-- We want BAM V1 output. -->
        <version>1</version>
        <!-- The BAM files will be called "example.bam" and saved in the current directory. -->
        <file>example.bam</file>
    </output>

    <input>
        <!-- Frames are imported from a static list of graphics files. Alternatively you can initialize
             a file sequencer to import graphics files based on parameters. -->
        <static>true</static>
        <files>
            <!-- Our list of graphics files. They are added to the BAM file in the same order as listed. -->
            <path>./files/frame00.png</path>
            <path>./files/frame01.png</path>
        </files>
    </input>

    <!-- General BAM settings -->
    <settings>
        <!-- Center positions for our frames are optional. Omitted entries will use the center position of
             the last defined entry or [0,0] if no entries are available. -->
        <center>14,16</center>
        <center>0,0</center>
        <!-- We define a single BAM cycle with our two frames. -->
        <sequence>0,1</sequence>
    </settings>

    <bamv1>
        <!-- Generate a compressed BAMC output file. -->
        <compress>true</compress>
        <!-- -1 indicates to apply RLE-encoding to BAM frames only if it yields a smaller file size. -->
        <rle>-1</rle>
        <!-- Discard alpha from palette. This is useful if the BAM should be compatible with the
             classic Infinity Engine games. -->
        <alpha>false</alpha>
        <!-- Sort BAM palette by lightness in reversed order. -->
        <sortby>lightness_reversed</sortby>
    </bamv1>

    <!-- Filters are optional. -->
    <filters>
        <!-- Filter "gamma" takes a single option. -->
        <filter>
            <name>gamma</name>
            <option>
              <key>level</key>
              <value>1.3</value>
            </option>
        </filter>
    </filters>
</generator>

Assuming this file is called "example.xml", we can generate a BAM file with the following call (using Windows syntax):

  bamcreator.exe --verbose example.xml

Command line parameters are optional. Some options, such as --verbose or --log-style, control the feedback produces
during the conversion process. Other option can be used to override settings in the configuration file.

BAM Creator accepts configuration data from standard input. This can be useful if you want to generate configurations
on the fly. Using the example from above (Windows syntax again):

  type example.xml | bamcreator.exe --verbose -

This command prints the content of example.xml to standard output and pipes it to the BAM Creator. The trailing minus
sign signals the tool to accept data from standard input in place of a file.

A fully commented example.xml with all available settings can be found in the "examples" subfolder. You can also find
example.json, which contains the same configuration in JSON format. (See notes in commented example.xml about JSON file
format limitations.)


Building from source
~~~~~~~~~~~~~~~~~~~~

The tool requires the Go compiler suite (https://golang.org/) to be installed. Code dependencies are automatically
resolved when using the included build scripts. However, the packages go-imagequant
(https://github.com/InfinityTools/go-imagequant) and go-squish (https://github.com/InfinityTools/go-squish) may have
to be prepared manually before they can be used in the build process.

Install the upx packer (https://github.com/upx/upx/releases) if you want to make use of the "--compress" option
of the build scripts.

Use the provided build scripts ("build.cmd" for Windows, "build.sh" for Linux and macOS) to build the binaries.
You can specify the build target and a number of options to influence the build process.

Syntax for build.sh (the same syntax applies to build.cmd):

Usage: build.sh [options] [target]

Options:
  --debug         Don't strip debugging symbols
  --nodeps        Don't check dependencies
  --update        Force updating dependencies
  --compress      Compress the binary with upx if available
  --help          This help

Available targets:
  bamcreator (default)

The resulting binary will be placed into the folder "./bin/$GOOS/$GOARCH" (e.g. "bin/darwin/amd64" for macOS).


Alternatively you can build binaries directly. Go into the subdirectory "bamcreator" and call "go build".


License
~~~~~~~

BAM Creator is released under the BSD 2-clause license. See LICENSE for more details.
