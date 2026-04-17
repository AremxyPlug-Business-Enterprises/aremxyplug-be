package airtime

import (
	"context"

	"github.com/aremxyplug-be/db/models/telcom"
)

func (a *AirtimeConn) SaveRecipient(ctx context.Context, userID string, data telcom.Recipient) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := a.db.SaveTelcomRecipient(ctx, userID, data); err != nil {
		return err
	}

	return nil
}

func (a *AirtimeConn) GetRecipients(ctx context.Context, username string) (telcom.TelcomRecipient, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	resp, err := a.db.GetTelcomRecipients(ctx, username)
	if err != nil {
		return telcom.TelcomRecipient{}, err
	}

	return resp, nil
}

func (a *AirtimeConn) UpdateRecipient(ctx context.Context, userID string, data telcom.Recipient) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := a.db.EditTelcomRecipient(ctx, userID, data); err != nil {
		return err
	}

	return nil
}

func (a *AirtimeConn) DeleteRecipient(ctx context.Context, recipientID int, userID string) error {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := a.db.DeleteTelcomRecipient(ctx, recipientID, userID); err != nil {
		return err
	}

	return nil
}
