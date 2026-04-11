CREATE UNIQUE INDEX IF NOT EXISTS wifi_bssid_unique ON wifi (bssid);
CREATE UNIQUE INDEX IF NOT EXISTS ip_address_unique ON ip (address);