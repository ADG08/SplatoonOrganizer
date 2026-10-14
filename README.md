# SplatoonOrganizer

Un bot Discord conçu pour coordonner les sessions de jeu Splatoon en collectant et affichant les disponibilités hebdomadaires des membres d'un serveur. Une seule instance du bot peut servir plusieurs serveurs : chacun a sa configuration, ses dispos et son message de la semaine.

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
| `DISCORD_GUILD_ID` | Non | Ancien serveur unique : ses commandes « de serveur » sont retirées au démarrage (les commandes sont désormais globales) |
| `DISCORD_CHANNEL_ID` | Non | Salon de ce serveur, repris dans sa configuration s'il n'en a pas encore |
| `DATABASE_URL` | Oui | URL de connexion PostgreSQL (local Docker: `...@db-dev:5432/...`) |
| `RUN_WEEKLY_ON_START` | Non | Mettre `1` pour republier dans tous les serveurs au démarrage (tests) |

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
| `/help` | Admin | Ouvre un panneau de configuration moderne avec sélecteurs interactifs (salon, rôle à ping, créneaux, affichage, langue et publication). |
| `/post-dispos` | Tous | Déclenche manuellement la publication du message hebdomadaire. |
| `/set-message-channel channel:<#salon>` | Admin | Définit le salon où le message hebdomadaire est publié. |
| `/set-role-to-ping role:<@role>` | Admin | Définit le rôle à mentionner lors de la publication. |

### Interactions (boutons / menus)

Lorsque le message hebdomadaire est publié, un bouton **✏️ Mes dispos** est affiché. En cliquant dessus :

1. Des menus déroulants privés (éphémères) s'ouvrent, un par jour de la semaine (Lundi–Dimanche).
2. Pour chaque jour, l'utilisateur peut sélectionner **Après-midi**, **Soir**, les deux, ou aucun (en mode **Par heure** : n'importe quelle combinaison d'heures, affichées à son heure).
3. Les choix sont enregistrés en base de données et le tableau public est mis à jour en temps réel.

### Créneaux et affichage

Depuis `/help` → **🕒 Créneaux, affichage, langue**, un admin choisit :

- la **langue** du bot sur le serveur : **Français** (par défaut) ou **English**. Tout le bot suit ce choix (message de la semaine, menus, réponses privées, `/help`). Seules les descriptions des commandes slash suivent la langue de l'application Discord de chaque membre, Discord ne permettant pas de les régler par serveur ;

- le **découpage** : **Après-midi / Soir** (par défaut) ou **Par heure**, un créneau par heure entre une première heure et une heure de fin (heure de Paris, 20h → minuit par défaut ; ces deux champs sont ignorés en « Après-midi / Soir ») ;
- l'**affichage** du message de la semaine :
  - **Classique** (par défaut) : l'embed d'origine avec le tableau en texte ;
  - **Image** : la grille dessinée en image (PNG générée par le bot, cases colorées selon le nombre de dispos) dans un message Components V2, avec un menu **👀 Voir qui est dispo…** qui affiche en privé qui est dispo un jour donné.

Changer le découpage réinitialise les dispos de la semaine en cours ; changer l'affichage ou la langue republie le message. Le réglage est stocké par serveur dans `guild_config`. La grille publique est à l'heure de Paris (Discord ne convertit pas les heures dans un bloc de code ni dans une image) ; les listes « Voir les dispos » utilisent les timestamps Discord, affichés dans le fuseau de chacun.

### Fuseaux horaires (mode Par heure)

Le bot ne peut pas connaître le fuseau d'un membre (Discord ne le transmet pas). Au premier clic sur **✏️ Mes dispos** ou **🌍 Mon fuseau**, il lui demande son décalage UTC (deux menus : UTC-12 → UTC-1 et UTC+0 → UTC+14, demi-heures comprises, chacun avec l'heure actuelle), puis le retient (table `user_timezone`) :

- les menus **✏️ Mes dispos** affichent les créneaux à l'heure du membre (ex. « 14h – 15h » à Montréal pour 20h à Paris, « Mar 3h – 4h » à Tokyo) ;
- le bouton **🌍 Mon fuseau** lui renvoie en privé la grille de la semaine recalculée à son heure, avec un bouton **⚙️ Modifier mon fuseau** ;
- le message public reste à l'heure de Paris, puisqu'il est partagé par tous.

Un décalage fixe ne suit pas l'heure d'été : le membre le met à jour avec **⚙️ Modifier mon fuseau** quand l'heure change.

## Planification automatique

Le bot publie automatiquement le message de disponibilités dans chaque serveur qui a configuré un salon via `/help`. À chaque publication, le message de la semaine précédente est supprimé.

Depuis `/help` → **📅 Publication**, un admin choisit quand (heure de Paris) :

- **Lundi** (par défaut, à 10h) : le message porte sur la semaine qui commence ;
- **Dimanche** : le message porte sur la semaine qui suit, pour que chacun remplisse ses dispos avant le lundi.

Une semaine va toujours du lundi au dimanche. Les clics comptent pour la semaine du dernier message publié : avec une publication le dimanche à 18h, les dispos cochées à partir de 18h vont dans la semaine suivante.

Le bot vérifie toutes les heures quels serveurs ont atteint leur heure de publication, ainsi qu'au démarrage : une publication manquée pendant un redémarrage est rattrapée dès que le bot revient.

## Plusieurs serveurs

Il suffit d'inviter le bot sur un serveur : les commandes slash sont globales. Un admin configure ensuite le salon (et éventuellement le rôle, les créneaux, l'affichage, la langue et la publication) avec `/help`. Le message de la semaine y est publié dans l'heure qui suit. Les dispos, les « pas dispo » et le message de la semaine sont propres à chaque serveur ; le fuseau horaire d'un membre est commun à tous les serveurs.

Pour rester réactif quand plusieurs membres cliquent en même temps, le message de la semaine d'un serveur est mis à jour tout de suite au premier clic, puis au plus une fois par seconde : les clics suivants sont regroupés.

Pour republier dans tous les serveurs au démarrage (utile pour les tests), utiliser `RUN_WEEKLY_ON_START=1`.

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
│   │   │   └── scheduler/       # Publication hebdomadaire (vérifiée toutes les heures)
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
-- Dernier message de la semaine publié par serveur
CREATE TABLE weekly_messages (
    guild_id   TEXT PRIMARY KEY,
    week       TEXT NOT NULL,  -- Clé de semaine ISO (ex: "2026-W10")
    channel_id TEXT NOT NULL,
    message_id TEXT NOT NULL
);

-- Stocke les disponibilités des utilisateurs
CREATE TABLE dispos (
    guild_id   TEXT     NOT NULL,
    user_id    TEXT     NOT NULL,
    day_index  SMALLINT NOT NULL,  -- 0=Lundi … 6=Dimanche
    slot_index SMALLINT NOT NULL,  -- Après-midi/Soir : 0, 1 · Par heure : 0 = première heure, 1 = +1h…
    week       TEXT     NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT dispos_unique_per_week UNIQUE (guild_id, week, user_id, day_index, slot_index)
);

-- Configuration par serveur Discord
CREATE TABLE guild_config (
    guild_id   TEXT PRIMARY KEY,
    channel_id TEXT,  -- Salon de publication
    role_id    TEXT,  -- Rôle à mentionner
    slot_mode       TEXT,      -- NULL/'halfday' = Après-midi/Soir, 'hourly' = par heure
    slot_start_hour SMALLINT,  -- mode hourly : première heure (0-23)
    slot_end_hour   SMALLINT,  -- mode hourly : fin du dernier créneau (1-24, 24 = minuit)
    display_style   TEXT,      -- NULL/'classic' = embed texte, 'image' = grille en image
    language        TEXT,      -- NULL/'fr' = français, 'en' = anglais
    publish_weekday SMALLINT,  -- NULL = lundi · 0 = dimanche (semaine qui suit), 1 = lundi
    publish_hour    SMALLINT   -- NULL = 10h · heure de publication (0-23, heure de Paris)
);

-- Fuseau horaire choisi par chaque membre (mode Par heure)
CREATE TABLE user_timezone (
    user_id    TEXT PRIMARY KEY,
    timezone   TEXT NOT NULL,  -- "UTC+2" ou nom IANA ("Europe/Paris")
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## Développement

### Régénérer le code sqlc

Après modification des requêtes SQL dans `internal/db/queries/` :

```bash
sqlc generate
```

### Lancer les tests

```bash
go test ./...
```

Les tests du dépôt PostgreSQL ont besoin d'une base : ils créent un schéma temporaire, y appliquent les migrations, puis le suppriment. Avec la base de dev lancée :

```bash
TEST_DATABASE_URL=postgres://splatoon:splatoon@localhost:5432/splatoon?sslmode=disable go test ./...
```

Sans `TEST_DATABASE_URL`, ils sont ignorés.

### Appliquer les migrations manuellement (local dev)

```bash
docker compose run --rm migrate-dev
```
