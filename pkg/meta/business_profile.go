package meta

import (
	"context"
	"encoding/json"
	"fmt"
)

// BusinessProfile mirrors the writable fields of Meta's WhatsApp Business
// Profile API (`/{phone-number-id}/whatsapp_business_profile`). Field names
// and constraints match the Graph API: about ≤139 chars, address ≤256,
// description ≤512, email ≤128, websites ≤2 URLs, vertical from the enum.
type BusinessProfile struct {
	MessagingProduct string   `json:"messaging_product,omitempty"`
	About            string   `json:"about,omitempty"`
	Address          string   `json:"address,omitempty"`
	Description      string   `json:"description,omitempty"`
	Email            string   `json:"email,omitempty"`
	Websites         []string `json:"websites,omitempty"`
	Vertical         string   `json:"vertical,omitempty"`
}

// GetBusinessProfile fetches the WhatsApp business profile of a phone number.
// Returned as a map so read-only fields (profile_picture_url, etc.) survive.
func (c *Client) GetBusinessProfile(ctx context.Context, accessToken, phoneNumberID string) (map[string]any, error) {
	body, err := c.graphGet(ctx, accessToken,
		phoneNumberID+"/whatsapp_business_profile",
		"about,address,description,email,profile_picture_url,websites,vertical")
	if err != nil {
		return nil, err
	}

	var res struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(body, &res); err != nil {
		return nil, fmt.Errorf("meta: parse business profile: %w", err)
	}
	if len(res.Data) == 0 {
		return map[string]any{}, nil
	}
	return res.Data[0], nil
}

// UpdateBusinessProfile writes the given profile fields to a phone number's
// WhatsApp business profile. Only non-empty fields are sent.
func (c *Client) UpdateBusinessProfile(ctx context.Context, accessToken, phoneNumberID string, profile *BusinessProfile) error {
	if profile.MessagingProduct == "" {
		profile.MessagingProduct = "whatsapp"
	}

	payload := map[string]any{
		"messaging_product": profile.MessagingProduct,
	}
	if profile.About != "" {
		payload["about"] = profile.About
	}
	if profile.Address != "" {
		payload["address"] = profile.Address
	}
	if profile.Description != "" {
		payload["description"] = profile.Description
	}
	if profile.Email != "" {
		payload["email"] = profile.Email
	}
	if len(profile.Websites) > 0 {
		payload["websites"] = profile.Websites
	}
	if profile.Vertical != "" {
		payload["vertical"] = profile.Vertical
	}

	_, err := c.graphPost(ctx, accessToken, phoneNumberID+"/whatsapp_business_profile", payload)
	return err
}