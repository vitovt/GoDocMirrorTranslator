package prompts

import "fmt"

func DocumentAnalysis(sourceLanguage, targetLanguage string, imageWidth, imageHeight int) string {
	return fmt.Sprintf(`Analyze the document image and return only JSON that matches the provided schema.

Task:
- Read handwritten or mixed handwritten/printed %s text from the image.
- Translate each text block into %s.
- Preserve the rough visual layout and column structure.
- Represent unreadable text blocks with source_text set to "[unreadable]".
- Do not hallucinate missing text.

Coordinate rules:
- Use the original source image pixel space.
- The image width is %d pixels and the image height is %d pixels.
- The coordinate origin is the top-left corner.
- x and y are the top-left position of each block.
- width and height must be non-negative pixel sizes.

Output rules:
- Return only valid JSON for the schema.
- Use one block per distinct visual text area.
- Include confidence when you can estimate it in the [0,1] range.
- Keep translated_text concise and faithful to the source block.`, sourceLanguage, targetLanguage, imageWidth, imageHeight)
}
