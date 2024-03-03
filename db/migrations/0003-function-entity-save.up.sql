-- UPSERT for an entity record.
-- It handles unique violation exceptions on both name and uuid, since it's not
-- possible to define multiple ON CONFLICT DO UPDATE blocks.
CREATE OR REPLACE FUNCTION entity_save(
    created_at_ timestamp, updated_at_ timestamp, uuid_ text, type_ entity_type, name_ text, description_ text, location_ text
) RETURNS setof entity
LANGUAGE plpgsql AS $$
  BEGIN
    INSERT INTO entity (created_at, uuid, type, name, description, location)
    VALUES (created_at_, uuid_, type_, name_, description_, location_);
    RETURN QUERY SELECT * FROM entity WHERE name = name_;
  EXCEPTION
    WHEN unique_violation THEN
      UPDATE entity
      SET (updated_at, type, description, location) =
          (updated_at_, type_, description_, location_)
      WHERE name = name_;
      RETURN QUERY SELECT * FROM entity WHERE name = name_;
  END;
$$
