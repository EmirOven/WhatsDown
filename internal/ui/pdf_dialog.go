package ui

import "postbetasoftware.com/whatsdown/internal/store"

func (c *MainController) openPDFPreview(msg store.StoredMessage, _ string) {
	c.openMediaViewer(msg)
}
