package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
)

const (
	CONVERT_TYPE_IMAGE_PVR   = 1
	CONVERT_TYPE_IMAGE_PVRGZ = 2
	CONVERT_TYPE_IMAGE_WEBP  = 3
	CONVERT_TYPE_FFMPEG      = 4
)

var PVR_TOOL_PATH = "PVRTexToolCLI"
var FFMPEG_TOOL_PATH = "ffmpeg"
var WEBP_TOOL_PATH = "cwebp"


func initializeToolsPathes() {
    // PVR
    pvrPath := os.Getenv("PVR_TOOL_PATH")
    if len(pvrPath) > 0 {
        PVR_TOOL_PATH = pvrPath
    }
    // FFMPEG
    ffmpegPath := os.Getenv("FFMPEG_TOOL_PATH")
    if len(ffmpegPath) > 0 {
        FFMPEG_TOOL_PATH = ffmpegPath
    }
    // FFMPEG
    webpPath := os.Getenv("WEBP_TOOL_PATH")
    if len(webpPath) > 0 {
		WEBP_TOOL_PATH = webpPath
    }
}

func convertFile(srcFilePath, resultFile, uuid string, convertType byte, paramsStr string) error {
	// Convert file
	switch convertType {
	case CONVERT_TYPE_IMAGE_PVR:
		// Params examples
		// -f PVRTC2_4 -dither -q pvrtcbest
		commandText := fmt.Sprintf("%s %s -i %s -o %s", PVR_TOOL_PATH, paramsStr, srcFilePath, resultFile)
		command := exec.Command("bash", "-c", commandText)
		err := command.Run()
		if err != nil {
			output, _ := command.CombinedOutput()
            log.Printf("Error exec command '%s': %s", commandText, string(output))
		}
		return err
	case CONVERT_TYPE_IMAGE_PVRGZ:
		// Params examples
		// -f r8g8b8a8 -dither -q pvrtcbest
		// -f r4g4b4a4 -dither -q pvrtcbest
		tempFileName := os.TempDir() + "out_" + uuid + ".pvr"
		convertCommandText := fmt.Sprintf("%s %s -i %s -o %s; gzip -f --suffix gz -9 %s", PVR_TOOL_PATH, paramsStr, srcFilePath, tempFileName, tempFileName)
		command := exec.Command("bash", "-c", convertCommandText)
		err := command.Run()
		if err != nil {
			output, _ := command.CombinedOutput()
            log.Printf("Error exec command '%s': %s", convertCommandText, string(output))
		}
		return err
    case CONVERT_TYPE_IMAGE_WEBP:
        // Params examples
        // -q 94
        commandText := fmt.Sprintf("%s %s %s -o %s", WEBP_TOOL_PATH, paramsStr, srcFilePath, resultFile)
        command := exec.Command("bash", "-c", commandText)
        err := command.Run()
        if err != nil {
            output, _ := command.CombinedOutput()
            log.Printf("Error exec command '%s': %s, %s", commandText, string(output))
        }
        return err
	case CONVERT_TYPE_FFMPEG:
		commandText := fmt.Sprintf("%s -i %s -y %s %s", FFMPEG_TOOL_PATH, srcFilePath, paramsStr, resultFile)
		command := exec.Command("bash", "-c", commandText)
		err := command.Run()
		if err != nil {
			output, _ := command.CombinedOutput()
            log.Printf("Error exec command '%s': %s", commandText, string(output))
		}
		return err
	}
	return errors.New("Invalid convert type")
}
