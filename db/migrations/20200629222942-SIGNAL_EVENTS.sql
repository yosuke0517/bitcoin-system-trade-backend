
-- +migrate Up
CREATE TABLE IF NOT EXISTS `SIGNAL_EVENTS` (
  `time` TIMESTAMP PRIMARY KEY NOT NULL,
  `product_code` VARCHAR(50),
  `side` VARCHAR(10),
  `price` float,
  `size` float
);
-- +migrate Down
DROP TABLE IF EXISTS `SIGNAL_EVENTS`;