DROP TRIGGER IF EXISTS registrations_sync_attendees_active ON registrations;
DROP FUNCTION IF EXISTS registrations_sync_attendees_active();
DROP TRIGGER IF EXISTS registrations_touch_status_changed_at ON registrations;
DROP FUNCTION IF EXISTS registrations_touch_status_changed_at();
DROP INDEX IF EXISTS registrations_status_created_at_idx;
DROP INDEX IF EXISTS attendees_email_active_key;
