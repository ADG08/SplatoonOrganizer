# SplatoonOrganizer

Un bot Discord conçu pour coordonner les sessions de jeu Splatoon en collectant et affichant les disponibilités hebdomadaires des membres d'un serveur.

## Fonctionnement

Chaque semaine, le bot publie automatiquement un message dans un salon Discord configuré. Les membres cliquent sur un bouton pour indiquer en privé leurs créneaux de disponibilité (après-midi / soir, pour chaque jour de la semaine). Le bot affiche un tableau récapitulatif en temps réel pour permettre aux organisateurs de voir quand le plus de joueurs sont disponibles.

## Stack technique

| Couche | Technologie |
|---|---|
| Langage | Go 1.26 |
| Bot Discord | `github.com/bwmarrin/discordgo` v0.29.0 |
| Base de données | PostgreSQL 16 |
| Driver BDD | `github.com/jackc/pgx/v5` v5.8.0 (pgxpool) |
| Génération de requêtes | `sqlc` v1.30.0 |
| Scheduler | `github.com/robfig/cron/v3` v3.0.1 |
| Conteneurisation | Docker + Docker Compose |
| Migrations | `migrate/migrate` v4.17.0 |

L'architecture suit le pattern **Hexagonal / Clean Architecture** avec une séparation claire entre le domaine, les cas d'usage (application) et les adaptateurs (Discord + Postgres).

## Prérequis

- [Docker](https://docs.docker.com/get-docker/) et [Docker Compose](https://docs.docker.com/compose/)
- Un bot Discord configuré sur le [portail développeur Discord](https://discord.com/developers/applications)

## Installation et lancement

### 1. Configurer les variables d'environnement

```bash
cp .env.example .env
```

Renseigner ensuite les valeurs dans `.env` (environnement local `dev`) :

| Variable | Requis | Description |
|---|---|---|
| `DISCORD_TOKEN` | Oui | Token du bot Discord |
| `DISCORD_CLIENT_ID` | Oui | ID de l'application Discord |
| `DISCORD_GUILD_ID` | Oui | ID du serveur Discord (guild) |
| `DISCORD_CHANNEL_ID` | Non | Salon par défaut pour les publications |
| `DATABASE_URL` | Oui | URL de connexion PostgreSQL (local Docker: `...@db-dev:5432/...`) |
| `CRON_SCHEDULE` | Non | Expression cron (défaut : `0 10 * * MON`) |
| `RUN_WEEKLY_ON_START` | Non | Mettre `1` pour publier au démarrage |

### 2. Lancer le projet en local (dev)

```bash
docker compose up -d --build
```

Docker Compose démarre automatiquement trois services `dev` :

- **`db-dev`** — PostgreSQL 16 avec un volume persistant local
- **`migrate-dev`** — Applique les migrations sur la base de données locale
- **`bot-dev`** — Compile et lance le bot Go

### Déploiement sur Coolify (prod)

Sur Coolify, ne comptez pas sur un fichier `.env` présent dans le repo : configurez les variables d'environnement directement dans l'UI Coolify.

Pour éviter que Coolify suive l'état des services `dev`, utilisez un fichier Compose dédié prod :

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

Ce fichier ne contient que `db-prod`, `migrate-prod` et `bot-prod`, ce qui maintient un état de santé cohérent côté Coolify.

### Arrêter l'environnement local (dev)

```bash
docker compose down
```

## Commandes Discord

### Commandes slash

| Commande | Permission | Description |
|---|---|---|
| `/dispos` | Tous | Affiche le détail des disponibilités de la semaine avec les mentions des joueurs (message éphémère). |
| `/post-dispos` | Tous | Déclenche manuellement la publication du message hebdomadaire. |
| `/set-message-channel channel:<#salon>` | Admin | Définit le salon où le message hebdomadaire est publié. |
| `/set-role-to-ping role:<@role>` | Admin | Définit le rôle à mentionner lors de la publication. |

### Interactions (boutons / menus)

Lorsque le message hebdomadaire est publié, un bouton **✏️ Mes dispos** est affiché. En cliquant dessus :

1. Des menus déroulants privés (éphémères) s'ouvrent, un par jour de la semaine (Lundi–Dimanche).
2. Pour chaque jour, l'utilisateur peut sélectionner **Après-midi**, **Soir**, les deux, ou aucun.
3. Les choix sont enregistrés en base de données et le tableau public est mis à jour en temps réel.

## Planification automatique

Le bot publie automatiquement le message de disponibilités chaque **lundi à 10h00 (Europe/Paris)** par défaut. À chaque publication, les anciens messages sont supprimés.

Le planning est configurable via la variable `CRON_SCHEDULE` (format cron standard). Pour déclencher une publication au démarrage (utile pour les tests), utiliser `RUN_WEEKLY_ON_START=1`.

## Structure du projet

```
SplatoonOrganizer/
├── cmd/
│   └── bot/
│       ├── main.go              # Point d'entrée
│       └── config.go            # Chargement de la configuration
│
├── internal/
│   ├── adapter/
│   │   ├── discord/
│   │   │   ├── bot.go           # Session Discord et dispatch des interactions
│   │   │   ├── registry.go      # Registre des commandes et handlers
│   │   │   ├── commands/        # Implémentation des commandes slash
│   │   │   ├── handlers/        # Handlers boutons / menus déroulants
│   │   │   └── scheduler/       # Tâche cron de publication hebdomadaire
│   │   │
│   │   └── persistence/
│   │       └── postgres/        # Implémentations des repositories PostgreSQL
│   │
│   ├── application/
│   │   ├── availability/        # Logique métier des disponibilités
│   │   └── guildconfig/         # Logique métier de la configuration du serveur
│   │
│   ├── domain/
│   │   ├── availability/        # Modèles de domaine (WeekKey, SlotCount…)
│   │   └── guildconfig/         # Modèle GuildConfig
│   │
│   └── db/                      # Code généré par sqlc (ne pas modifier)
│
├── migrations/                  # Fichiers de migration SQL
├── Dockerfile
├── docker-compose.yml           # Stack locale dev
├── docker-compose.prod.yml      # Stack prod pour Coolify
├── sqlc.yaml
└── .env.example
```

## Schéma de base de données

```sql
-- Stocke les messages publiés par semaine
CREATE TABLE sondage_messages (
    week       TEXT PRIMARY KEY,  -- Clé de semaine ISO (ex: "2026-W10")
    message_id TEXT NOT NULL      -- ID du message Discord
);

-- Stocke les disponibilités des utilisateurs
CREATE TABLE dispos (
    user_id    TEXT     NOT NULL,
    day_index  SMALLINT NOT NULL,  -- 0=Lundi … 6=Dimanche
    slot_index SMALLINT NOT NULL,  -- 0=Après-midi, 1=Soir
    week       TEXT     NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT dispos_unique_per_week UNIQUE (user_id, day_index, slot_index, week)
);

-- Configuration par serveur Discord
CREATE TABLE guild_config (
    guild_id   TEXT PRIMARY KEY,
    channel_id TEXT,  -- Salon de publication
    role_id    TEXT   -- Rôle à mentionner
);
```

## Développement

### Régénérer le code sqlc

Après modification des requêtes SQL dans `internal/db/queries/` :

```bash
sqlc generate
```

### Appliquer les migrations manuellement (local dev)

```bash
docker compose run --rm migrate-dev
```
