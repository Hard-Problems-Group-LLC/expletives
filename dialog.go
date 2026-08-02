package expletives

import "context"

// DialogOptions configures one ModalPanel-derived dialog container.
type DialogOptions struct {
	ModalPanelOptions
}

// Dialog is a copy-safe ModalPanel-derived container. Its children use the
// ordinary control and Layout APIs; Button Default and Cancel roles are scoped
// across the complete Dialog subtree.
type Dialog struct {
	ModalPanel
}

// NewDialog constructs one inactive Dialog as a direct App root child.
func NewDialog(parent Container, options DialogOptions) (*Dialog, error) {
	tx, err := transactionForParent(parent)
	if err != nil {
		return nil, err
	}
	dialog, err := tx.NewDialog(parent, options)
	if err != nil {
		return nil, err
	}
	if err := tx.Commit(context.Background()); err != nil {
		return nil, err
	}
	return dialog, nil
}

// NewDialog records construction of one provisional inactive Dialog.
func (t *Transaction) NewDialog(
	parent Container,
	options DialogOptions,
) (*Dialog, error) {
	return t.newDialog(parent, options, ControlDialog)
}

func (t *Transaction) newDialog(
	parent Container,
	options DialogOptions,
	kind ControlKind,
) (*Dialog, error) {
	modal, err := t.newModalPanel(
		parent,
		options.ModalPanelOptions,
		kind,
	)
	if err != nil {
		return nil, err
	}
	dialog := &Dialog{ModalPanel: *modal}
	modal.state.control = dialog
	modal.state.container = dialog
	return dialog, nil
}
