package ginupload

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/flytam/filenamify"
	"github.com/gin-gonic/gin"
)

type uploadForm struct {
	filename   string
	chunkIndex uint64
	chunks     uint64
}

func UploadHandler(uploadDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var form uploadForm
		c.Bind(&form)

		form.filename, _ = filenamify.Filenamify(form.filename, filenamify.Options{})
		if form.filename == "" {
			c.AbortWithStatusJSON(400, gin.H{
				"error": "filename is required",
			})
			return
		}

		file, _ := c.FormFile("file")
		// Assign a unique filename for each chunk.
		partFileName := fmt.Sprintf("%s.%d.part", form.filename, form.chunkIndex)
		partFilePath := filepath.Join(uploadDir, partFileName)

		// Save the uploaded chunk file.
		if err := c.SaveUploadedFile(file, partFilePath); err != nil {
			c.JSON(500, gin.H{"error": "failed to save file part"})
			return
		}

		// Merge all chunks if this is the last chunk.
		if form.chunkIndex == form.chunks-1 {
			mergeChunks(uploadDir, form.filename, form.chunks)
		}

		c.JSON(200, gin.H{
			"message":    "Chunk uploaded successfully",
			"chunkIndex": form.chunkIndex,
			"chunks":     form.chunks,
		})
	}
}

// Merge Chunks
func mergeChunks(uploadDir, fileName string, totalChunks uint64) {
	// Create target file
	finalFilePath := filepath.Join(uploadDir, fileName)
	finalFile, err := os.Create(finalFilePath)
	if err != nil {
		fmt.Println("Error creating final file:", err)
		return
	}
	defer finalFile.Close()

	// Merge the chunk files into the target file by order.
	for i := uint64(0); i < totalChunks; i++ {
		partFilePath := filepath.Join(uploadDir, fmt.Sprintf("%s.%d.part", fileName, i))
		partFile, err := os.Open(partFilePath)
		if err != nil {
			fmt.Println("Error opening part file:", err)
			return
		}
		defer partFile.Close()

		// Write the content of this chunk file into the target file.
		_, err = io.Copy(finalFile, partFile)
		if err != nil {
			fmt.Println("Error copying part file to final file:", err)
			return
		}

		// Delete the merged chunk file.
		partFile.Close()
		if err := os.Remove(partFilePath); err != nil {
			fmt.Println("Error removing part file:", err)
		}
	}

	fmt.Printf("File %s merged successfully\n", finalFilePath)
}

// Retrive uploading progress
func GetUploadProgressHandler(uploadDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		filename := c.DefaultQuery("filename", "")
		filename, _ = filenamify.Filenamify(filename, filenamify.Options{})
		if filename == "" {
			c.JSON(400, gin.H{"error": "Upload Filename is required"})
			return
		}

		// Count uploaded chunks.
		uploadedChunks := 0
		for i := 0; ; i++ {
			partFilePath := filepath.Join(uploadDir, fmt.Sprintf("%s.%d.part", filename, i))
			if _, err := os.Stat(partFilePath); os.IsNotExist(err) {
				break
			}
			uploadedChunks++
		}

		// Return uploading progress
		c.JSON(200, gin.H{
			"uploadedChunks": uploadedChunks,
		})
	}
}
