package airtime

import "github.com/aremxyplug-be/db/models/telcom"

func (a *AirtimeConn) SaveTelcomRecipient(data telcom.TelcomRecipient) error {
	if err := a.db.SaveTelcomRecipient(data); err != nil {
		return err
	}

	return nil
}

func (a *AirtimeConn) GetTelcomRecipients(username string) ([]telcom.TelcomRecipient, error) {
	resp, err := a.db.GetTelcomRecipients(username)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

func (a *AirtimeConn) UpdateTelcomRecipient(userID string, data telcom.TelcomRecipient) error {
	if err := a.db.EditTelcomRecipient(userID, data); err != nil {
		return err
	}

	return nil
}

func (a *AirtimeConn) DeleteTelcomRecipient(name, userID string) error {
	if err := a.db.DeleteTelcomRecipient(name, userID); err != nil {
		return err
	}

	return nil
}
