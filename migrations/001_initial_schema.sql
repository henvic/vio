-- Write your migrate up statements here

-- geolocation table
CREATE TABLE geolocation (
	ip_address cidr PRIMARY KEY,
	country_code text NOT NULL,
	country text NOT NULL,
	city text NOT NULL,
	-- Using text type for latitude and longitude for simplicity sake.
	-- Otherwise, using something like a PostGIS type for coordinates would be better.
	latitude text NOT NULL,
	longitude text NOT NULL,
	
	updated_at timestamp with time zone NOT NULL DEFAULT now()
);

COMMENT ON COLUMN geolocation.ip_address IS 'IP address for the geolocation data';

---- create above / drop below ----

-- Write your migrate down statements here. If this migration is irreversible
-- Then delete the separator line above.
DROP TABLE geolocation;
