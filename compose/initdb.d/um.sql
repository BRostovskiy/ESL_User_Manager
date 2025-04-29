--
-- PostgreSQL database dump
--

-- Dumped from database version 10.6
-- Dumped by pg_dump version 10.6

SET statement_timeout = 0;
SET lock_timeout = 0;
SET idle_in_transaction_session_timeout = 0;
SET client_encoding = 'UTF8';
SET standard_conforming_strings = on;
SELECT pg_catalog.set_config('search_path', '', false);
SET check_function_bodies = false;
SET client_min_messages = warning;
SET row_security = off;

--
-- Name: plpgsql; Type: EXTENSION; Schema: -; Owner: -
--

CREATE EXTENSION IF NOT EXISTS plpgsql WITH SCHEMA pg_catalog;


--
-- Name: EXTENSION plpgsql; Type: COMMENT; Schema: -; Owner: -
--

COMMENT ON EXTENSION plpgsql IS 'PL/pgSQL procedural language';


SET default_tablespace = '';

SET default_with_oids = false;

--
-- Name: users; Type: TABLE; Schema: public; Owner: -
--

CREATE TABLE public.users (
    id text NOT NULL,
    first_name text NOT NULL,
    last_name text NOT NULL,
    nickname text NOT NULL,
    password text NOT NULL,
    email text NOT NULL,
    country text NOT NULL,
    created_at timestamp without time zone,
    updated_at timestamp without time zone DEFAULT now()
);


--
-- Name: users email_uq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT email_uq UNIQUE (email);


--
-- Name: users id_uq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT id_uq UNIQUE (id);


--
-- Name: users nickname_uq; Type: CONSTRAINT; Schema: public; Owner: -
--

ALTER TABLE ONLY public.users
    ADD CONSTRAINT nickname_uq UNIQUE (nickname);


--
-- Name: country_idx; Type: INDEX; Schema: public; Owner: -
--

CREATE INDEX country_idx ON public.users USING btree (country);


--
-- PostgreSQL database dump complete
--

