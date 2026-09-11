-- Creates the kinara_events database for the shared event bus.
-- Run as the postgres superuser before applying schema migrations.
-- The schema is applied by V202609110052__Event_Bus__Init.sql via run-migrations.sh.
CREATE DATABASE kinara_events;
