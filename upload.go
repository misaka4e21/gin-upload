package ginupload

import (
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"

	"github.com/flytam/filenamify"
	"github.com/gin-gonic/gin"
	"github.com/gin-gonic/gin/binding"
)

type UploadForm struct {
	Filename   string                `form:"filename"`
	ChunkIndex uint64                `form:"chunk_index"`
	Chunks     uint64                `form:"chunks"`
	File       *multipart.FileHeader `form:"file"`
}

func UploadHandler(uploadDir string) gin.HandlerFunc {
	return func(c *gin.Context) {
		var form UploadForm
		if err := c.ShouldBindWith(&form, binding.FormMultipart); err != nil {
			fmt.Println(err.Error())
		}

		fmt.Println(form.Filename)
		form.Filename, _ = filenamify.Filenamify(form.Filename, filenamify.Options{})
		fmt.Println(form.Filename)
		if form.Filename == "" {
			c.AbortWithStatusJSON(400, gin.H{
				"error": "filename is required",
			})
			return
		}

		file := form.File
		// Assign a unique filename for each chunk.
		partFileName := fmt.Sprintf("%s.%d.part", form.Filename, form.ChunkIndex)
		partFilePath := filepath.Join(uploadDir, partFileName)

		// Save the uploaded chunk file.
		if err := c.SaveUploadedFile(file, partFilePath); err != nil {
			c.JSON(500, gin.H{"error": "failed to save file part"})
			return
		}

		// Merge all chunks if this is the last chunk.
		if form.ChunkIndex == form.Chunks-1 {
			mergeChunks(uploadDir, form.Filename, form.Chunks)
		}

		c.JSON(200, gin.H{
			"message":    "Chunk uploaded successfully",
			"chunkIndex": form.ChunkIndex,
			"chunks":     form.Chunks,
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
