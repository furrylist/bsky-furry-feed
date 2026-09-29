ALTER TABLE candidate_actors ADD COLUMN refuses_algorithmic_recommendations boolean DEFAULT false NOT NULL;
CREATE INDEX candidate_actors_refuses_algorithmic_recommendations_idx ON candidate_actors USING btree (
    refuses_algorithmic_recommendations
);
