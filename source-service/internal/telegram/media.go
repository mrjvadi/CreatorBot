package telegram

import "github.com/gotd/td/tg"

// mediaLocation extracts a downloadable file location, suggested file name,
// and MIME type from a message's media (document or photo).
func mediaLocation(media tg.MessageMediaClass) (tg.InputFileLocationClass, string, string, error) {
	switch m := media.(type) {
	case *tg.MessageMediaDocument:
		doc, ok := m.Document.(*tg.Document)
		if !ok {
			return nil, "", "", errNoDocument
		}
		name := "file"
		for _, attr := range doc.Attributes {
			if fa, ok := attr.(*tg.DocumentAttributeFilename); ok {
				name = fa.FileName
			}
		}
		loc := &tg.InputDocumentFileLocation{
			ID:            doc.ID,
			AccessHash:    doc.AccessHash,
			FileReference: doc.FileReference,
		}
		return loc, name, doc.MimeType, nil

	case *tg.MessageMediaPhoto:
		photo, ok := m.Photo.(*tg.Photo)
		if !ok {
			return nil, "", "", errNoPhoto
		}
		var largest *tg.PhotoSize
		for _, s := range photo.Sizes {
			if ps, ok := s.(*tg.PhotoSize); ok {
				largest = ps
			}
		}
		if largest == nil {
			return nil, "", "", errNoPhotoSize
		}
		loc := &tg.InputPhotoFileLocation{
			ID:            photo.ID,
			AccessHash:    photo.AccessHash,
			FileReference: photo.FileReference,
			ThumbSize:     largest.Type,
		}
		return loc, "photo.jpg", "image/jpeg", nil

	default:
		return nil, "", "", errUnsupportedMedia(media)
	}
}
