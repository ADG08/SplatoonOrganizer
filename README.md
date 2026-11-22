# Squad Planner

[![Soutenir le bot](https://img.shields.io/badge/Soutenir_le_bot-Revolut-0075EB?logo=revolut&logoColor=white)](https://revolut.me/adg01)

Un bot Discord de gestion d'équipes, pour tous les jeux : il collecte et affiche les disponibilités hebdomadaires des joueurs pour trouver quand jouer ensemble. Une seule instance du bot peut servir plusieurs serveurs, et chaque serveur peut avoir plusieurs **équipes** : chacune a ses réglages, ses dispos et son message de la semaine.

Né pour des équipes Splatoon, le projet garde son ancien nom technique : le dépôt et le module Go s'appellent toujours `SplatoonOrganizer`.

## Fonctionnement

Chaque semaine, le bot publie un message dans le salon de chaque équipe. Les membres cliquent sur **✏️ Mes dispos** pour indiquer en privé leurs créneaux, jour par jour : **Après-midi / Soir**, ou des créneaux d'une heure (20h → minuit par défaut). Le message affiche en temps réel combien de joueurs sont disponibles sur chaque créneau, pour trouver quand jouer.

Toutes les heures d'une équipe suivent **son fuseau horaire**, **Paris par défaut**, modifiable dans `/help` : créneaux, grille publique, semaines, heure de publication. Chaque membre peut en plus voir les créneaux à sa propre heure (voir [Fuseaux horaires](#fuseaux-horaires)).

## Soutenir le projet

Le bot est gratuit. Si tu veux participer aux frais d'hébergement : [revolut.me/adg01](https://revolut.me/adg01) 💸

## Stack technique

| Couche | Technologie |
|---|---|
| Langage | Go 1.26 |
| Bot Discord | `github.com/bwmarrin/discordgo` (v0.29, avec Components V2) |
| Base de données | PostgreSQL 16 |
| Driver BDD | `github.com/jackc/pgx/v5` v5.9 (pgxpool) |
| Génération de requêtes | `sqlc` v1.30.0 |
| Planification | bibliothèque standard (une vérification par quart d'heure) |
| Image de la grille | bibliothèque standard (`image`, `image/png`) + `golang.org/x/image` pour la police |
| Logs | `log/slog` (lignes `clé=valeur`) |
| Conteneurisation | Docker + Docker Compose, image `distroless/static` (~25 Mo, sans shell, utilisateur non root) |
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
| `DATABASE_URL` | Oui | URL de connexion PostgreSQL (local Docker : `...@db-dev:5432/...`) |
| `DISCORD_OWNER_ID` | Non | ID Discord du créateur du bot : il peut utiliser `/help` et `/annonce` sans être admin du serveur |
| `RUN_WEEKLY_ON_START` | Non | Mettre `1` pour republier le message de toutes les équipes au démarrage (tests) |
| `HEALTH_PORT` | Non | Port du healthcheck `/health` (défaut : `8080`) |
| `SUPPORT_URL` | Non | Page de dons : ajoute un bouton « 💸 Soutenir le bot » en bas de `/help` |
| `LOG_WEBHOOK_URL` | Non | Webhook d'un salon privé (Paramètres du salon → Intégrations → Webhooks) : les logs y sont aussi postés, en `[LOG]` (bot ajouté ou retiré d'un serveur, équipe créée ou supprimée, paramètres modifiés, message publié), `[WARNING]` et `[ERROR]` |

### 2. Lancer le projet en local (dev)

```bash
docker compose up -d --build
```

Docker Compose démarre automatiquement trois services `dev` :

- **`db-dev`** — PostgreSQL 16 avec un volume persistant local
- **`migrate-dev`** — Applique les migrations sur la base de données locale, puis s'arrête (`Exited (0)` est normal)
- **`bot-dev`** — Compile et lance le bot Go, une fois les migrations appliquées

Le bot lit sa configuration dans les variables d'environnement : Docker Compose les prend dans `.env`. Pour le lancer hors Docker, exportez-les d'abord dans le shell. Le `.env` n'est jamais envoyé à Docker (voir `.dockerignore`).

Utilisez un **bot de test** (un autre token que la prod) : deux processus avec le même token répondraient chacun aux clics.

### Arrêter l'environnement local (dev)

```bash
docker compose down
```

### Déploiement sur Coolify (prod)

Sur Coolify, ne comptez pas sur un fichier `.env` présent dans le repo : configurez les variables d'environnement directement dans l'UI Coolify.

Pour éviter que Coolify suive l'état des services `dev`, utilisez un fichier Compose dédié prod :

```bash
docker compose -f docker-compose.prod.yml up -d --build
```

Ce fichier ne contient que `db-prod`, `migrate-prod` et `bot-prod` : `migrate-prod` applique les migrations puis s'arrête, et le bot ne démarre qu'une fois qu'il a réussi. `exclude_from_hc: true` dit à Coolify de ne pas compter ce conteneur arrêté comme en mauvaise santé ; cette clé est propre à Coolify : `docker compose` seul la refuse.

Le healthcheck du bot appelle le bot lui-même (`/bot healthcheck`, qui interroge `/health`) : l'image ne contient ni shell ni `wget`.

Le bot doit tourner en **un seul exemplaire** : les équipes et les fuseaux des membres sont gardés en cache mémoire, et deux exemplaires publieraient chaque message deux fois.

## Ajouter le bot à un serveur

1. Sur le [portail développeur](https://discord.com/developers/applications) → l'application → **Bot**, activer **Public Bot** pour que d'autres que le créateur puissent l'inviter.
2. Inviter le bot avec ce lien (remplacer `CLIENT_ID`) :

   ```
   https://discord.com/oauth2/authorize?client_id=CLIENT_ID&scope=bot+applications.commands&permissions=268684304
   ```

   `268684304` = voir les salons, envoyer des messages, intégrer des liens, joindre des fichiers (l'image de la grille), lire l'historique, mentionner les rôles, plus **gérer les salons** et **gérer les rôles** (pour créer le salon et le rôle d'une équipe depuis `/help` ; sans elles, on choisit un salon et un rôle existants).
3. Le bot se présente (« 👋 Heeey, moi c'est … ! », avec son nom et son avatar) dans le salon système du serveur, ou à défaut dans le premier salon où il peut écrire, avec une checklist (`▱▱ 0/2` : créer une équipe, choisir son salon ; le rôle à mentionner y figure en option), un bouton **🚀 Lancer le setup** réservé aux admins et un bouton de langue pour choisir celle du serveur dès le début : en français par défaut, **🇬🇧 EN** le passe en anglais, puis **🇫🇷 FR** le remet en français. La checklist avance au fil de la configuration. Un serveur déjà connu du bot n'a pas de nouveau message : le message part uniquement à l'arrivée du bot, pas à ses redémarrages.
4. Un admin crée une équipe avec ce bouton ou `/help`, puis choisit son salon, son rôle et, si besoin, ses autres réglages. Le message de la semaine y est publié tout de suite.

## Qui peut faire quoi

| Action | Qui |
|---|---|
| `/help` (équipes et réglages) | Les membres ayant la permission **Administrateur** sur le serveur, et le créateur du bot (`DISCORD_OWNER_ID`) |
| `/annonce` | Les mêmes |
| Boutons et menus du message de la semaine | Tous les membres |

Les autres membres voient `/help` dans la liste des commandes, mais le bot leur répond qu'elle est réservée aux administrateurs.

## Commandes slash

| Commande | Description |
|---|---|
| `/help` | « Mes équipes » (message privé) : la liste des équipes du serveur, chacune avec ses réglages. Sans équipe, le setup guidé. |
| `/annonce titre:<texte> message:<texte> [equipe:<équipe>] [role:<@rôle>]` | Publie une annonce (embed) dans le salon d'une équipe, en mentionnant son rôle ou celui choisi. `equipe` est proposé en autocomplétion, et facultatif s'il n'y a qu'une équipe. `\n` dans le message fait un retour à la ligne. |

Les descriptions des commandes suivent la langue de l'application Discord de chaque membre (Discord ne permet pas de les régler par serveur).

## Le message de la semaine

| Bouton / menu | Effet (réponse privée) |
|---|---|
| **✏️ Mes dispos** | Un menu par jour pour cocher ses créneaux. Ouvrir ce menu annule un « pas dispo ». |
| **📋 Voir les dispos** (affichage classique) | Qui est dispo sur chaque créneau, les heures consécutives regroupées (« 20h – minuit »). |
| **👀 Voir qui est dispo…** (affichage image) | Qui est dispo un jour donné, ou toute la semaine. |
| **🚫 Pas dispo cette semaine** | Marque le membre indisponible pour la semaine et efface ses créneaux ; recliquer annule. |
| **🌍 Mon fuseau** (créneaux par heure) | La grille de la semaine recalculée à l'heure du membre. |

Les réponses privées indiquent l'équipe concernée : un membre peut être dans plusieurs, et remplit alors le message de chacune. Le message est mis à jour tout de suite au premier clic, puis au plus une fois par seconde quand plusieurs membres cliquent en même temps.

## Configuration avec `/help`

`/help` ouvre **👥 Mes équipes** : une ligne par équipe avec son état (✅ prête dès qu'elle a un salon, 🛠️ `▰▱ 1/2` en cours, ⏸️ en pause) et un bouton pour l'ouvrir, vert une fois l'équipe prête, plus **➕ Créer une équipe** (20 équipes au maximum par serveur). Sans équipe, le panneau affiche le setup guidé, **🚀 Créer ma première équipe** et le bouton de langue du serveur (**🇬🇧 EN** / **🇫🇷 FR**).

Le panneau d'une équipe montre où en est son setup (équipe créée, salon ; le rôle est suggéré, pas obligatoire) et ses réglages par section, chacune avec son bouton **✏️ Modifier** à droite. Tout est propre à l'équipe. En bas : **⬅️ Mes équipes**, **✏️ Renommer**, **📤 Publier maintenant** et **🗑️ Supprimer** (après confirmation ; ses dispos et son message partent avec elle, le salon et le rôle Discord restent).

- **📢 Salon et rôle** :
  - le **salon** où le message de la semaine est publié : un salon existant, ou le nom d'un salon que le bot crée ; en choisir un nouveau y publie le message tout de suite ;
  - le **rôle** mentionné à chaque publication, optionnel, existant ou créé par le bot (mentionnable) : le retirer arrête les mentions.
- **📅 Publication** : le **jour** et l'**heure** (heure de l'équipe, voir [Publication automatique](#publication-automatique)).
- **🕒 Créneaux** : le **découpage**, **Après-midi / Soir** (par défaut) ou **Par heure**, un créneau par heure entre une première heure et une heure de fin (heure de l'équipe, 20h → minuit par défaut ; ces deux champs sont ignorés en « Après-midi / Soir »). La fin peut aller **jusqu'à 5h le lendemain** (ex. 15h → 2h) : les heures après minuit restent dans la ligne du jour où la soirée a commencé (« Lundi 1h » = la nuit de lundi à mardi), et les listes « Voir les dispos » affichent leur vraie date. Changer le découpage réinitialise les dispos de la semaine en cours.

  En mode **Par heure**, le champ **Heures des membres** choisit entre **🏠 Tout le monde à l'heure de l'équipe** (par défaut : une équipe dont les joueurs sont dans le même pays) et **🌍 International** (chacun indique son fuseau et voit les créneaux à son heure, voir [Le fuseau des membres](#le-fuseau-des-membres)).
- **🌍 Fuseau horaire** : le fuseau de l'équipe, **Europe/Paris par défaut**. Le panneau se change en sélecteur (décalage UTC actuel, puis pays), puis revient avec le fuseau enregistré. En mode **Par heure**, changer de fuseau déplace les créneaux : les dispos de la semaine sont réinitialisées et le message republié.
- **🎨 Message** :
  - l'**affichage** : **Classique** (par défaut, l'embed d'origine avec le tableau en texte) ou **Image** (la grille dessinée par le bot, cases colorées selon le nombre de dispos) ;
  - la **langue** du bot pour l'équipe : **Français** (par défaut) ou **English** ; tout ce qui concerne l'équipe suit ce choix (message de la semaine, menus, réponses privées, son panneau). « Mes équipes », le message d'accueil et les nouvelles équipes suivent la **langue du serveur**, choisie dès le début avec le bouton 🇬🇧 EN / 🇫🇷 FR du message d'accueil ou du setup de `/help`. Sans choix, c'est la langue de la première équipe, puis, sans équipe, celle de l'application Discord de l'admin (le message d'accueil, lui, est en français) ;
  - **Case verte à partir de** (affichage image) : **Automatique** (par défaut, les couleurs dépendent du meilleur créneau de la semaine : vert dès 75 % de ce maximum, jaune dès 40 %) ou un nombre de joueurs, de 1 à 20. Avec 4 par exemple : 🟩 4 joueurs et plus, 🟨 2 à 3, ⬛ 1 ; la légende de l'image affiche alors ces seuils. Ce réglage redessine le message sans le republier.

  Changer l'affichage ou la langue republie le message.

## Publication automatique

À chaque publication, le message de la semaine précédente est supprimé. Une semaine va toujours du lundi au dimanche, et le jour de publication se choisit dans `/help` → **📅 Publication** :

- **Lundi** (par défaut, à 10h) : le message porte sur la semaine qui commence ;
- **Dimanche** : le message porte sur la semaine qui suit, pour que chacun remplisse ses dispos avant le lundi.

Les clics comptent pour la semaine du dernier message publié : avec une publication le dimanche à 18h, les dispos cochées à partir de 18h vont dans la semaine suivante.

En mode **Par heure**, les créneaux qui commencent avant la publication sont **fermés** : avec une publication le lundi à 10h, ceux du lundi avant 10h. Ils n'apparaissent pas dans **✏️ Mes dispos** (le menu d'un jour entièrement fermé est grisé), sont refusés s'ils sont envoyés quand même, et sont barrés dans la grille (hachurés dans l'image, `×` dans le tableau texte), avec une mention dans la légende. Avec une publication le dimanche, aucun créneau n'est fermé.

Le bot vérifie tous les quarts d'heure quelles équipes ont atteint leur heure de publication (dans leur fuseau), ainsi qu'au démarrage : une publication manquée pendant un redémarrage est rattrapée dès que le bot revient.

## Fuseaux horaires

### Le fuseau de l'équipe

Chaque équipe a un fuseau horaire, **Europe/Paris par défaut**, choisi dans `/help` → **🌍 Fuseau horaire**. Ses heures le suivent toutes : les créneaux (« 20h → minuit »), la grille publique, le début et la fin des semaines, l'heure de publication. Le fuseau d'un pays suit automatiquement l'heure d'été et d'hiver.

Le bot vérifie tous les quarts d'heure quelles équipes ont atteint leur heure de publication : certains fuseaux sont décalés d'une demi-heure ou d'un quart d'heure (Inde, Népal).

### Le fuseau des membres

La grille publique est à l'heure de l'équipe : elle est partagée par tous, et Discord ne convertit pas les heures dans un texte ni dans une image. En mode **Par heure**, si l'équipe est réglée en **🌍 International** (`/help` → **🕒 Créneaux** → **Heures des membres**), chaque membre peut voir les créneaux à son heure.

Réglée sur **🏠 Tout le monde à l'heure de l'équipe** (par défaut), le bot ne demande rien aux membres et n'affiche pas **🌍 Mon fuseau** : tout est à l'heure de l'équipe. Les listes **📋 Voir les dispos** et **👀 Voir qui est dispo…** s'affichent quand même à l'heure de chacun, Discord convertissant lui-même leurs heures.

Discord ne transmet pas le fuseau d'un membre. Au premier clic sur **✏️ Mes dispos** ou **🌍 Mon fuseau**, le bot le lui demande en deux étapes :

1. son **décalage UTC** (UTC-12 → UTC+14, demi-heures comprises), chaque option affichant l'heure qu'il y est en ce moment ;
2. son **pays**, parmi ceux qui ont ce décalage en ce moment. Le bot retient le fuseau du pays (`Europe/Paris`, `America/Toronto`…) : les **passages à l'heure d'été et d'hiver sont suivis automatiquement**. Une dernière option garde un décalage fixe pour les pays absents de la liste ; celui-là ne suit pas les changements d'heure.

Ensuite :

- les menus **✏️ Mes dispos** affichent les créneaux à l'heure du membre (ex. pour une équipe à Paris, « 14h – 15h » à Montréal pour 20h, « 3h – 4h (lendemain) » à Tokyo ; les heures de la nuit suivante sont marquées « (lendemain) ») ;
- **🌍 Mon fuseau** renvoie la grille de la semaine recalculée à son heure ;
- **⚙️ Modifier mon fuseau** permet d'en changer (en cas de déménagement ou de voyage, par exemple).

Le fuseau d'un membre est commun à toutes les équipes et à tous les serveurs.

## Plusieurs serveurs, plusieurs équipes

Les commandes slash sont globales : il suffit d'inviter le bot (voir [Ajouter le bot à un serveur](#ajouter-le-bot-à-un-serveur)). Une équipe appartient à un serveur ; ses réglages, ses dispos, ses « pas dispo » et son message de la semaine lui sont propres. Un joueur dans deux équipes remplit donc deux messages. Le rôle d'une équipe sert seulement à la mentionner : qui peut répondre dépend des permissions de son salon.

### Équipes actives et en pause

Une équipe passe **en pause** (plus aucune publication) quand :

- le bot est retiré du serveur (toutes ses équipes) ;
- Discord répond que le salon n'existe plus ou que le bot n'y a plus accès (permissions retirées).

Ses réglages sont gardés. Elle redevient **active** quand :

- le bot est ajouté de nouveau au serveur (ou le retrouve en redémarrant) ;
- un admin choisit à nouveau le salon dans `/help` → **📢 Salon et rôle**, ou réussit **📤 Publier maintenant** ; le panneau signale un salon en pause par un ⚠️.

### Passage à plusieurs équipes

La migration `013_teams` donne à chaque serveur déjà configuré une équipe **Équipe principale** (**Main team** s'il est en anglais) qui reprend tous ses réglages, ses dispos et son message de la semaine. Les boutons de ce message, publié avant les équipes, restent valables : ils sont rattachés à cette équipe jusqu'à la prochaine publication. Pour que le bot puisse créer des salons et des rôles sur un serveur existant, il faut lui ajouter **Gérer les salons** et **Gérer les rôles**, ou le réinviter avec le lien ci-dessus.

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
│   │   │   ├── helppanel.go     # Panneaux /help : Mes équipes, réglages d'une équipe
│   │   │   ├── welcome.go       # Message d'accueil
│   │   │   ├── commands/        # Commandes slash (/help, /annonce)
│   │   │   ├── handlers/        # Boutons, menus, formulaires, arrivée/départ du bot
│   │   │   ├── weekly/          # Message de la semaine : construction, publication, mise à jour
│   │   │   ├── gridimage/       # Grille dessinée en PNG (affichage Image)
│   │   │   └── scheduler/       # Publication automatique (vérifiée tous les quarts d'heure)
│   │   │
│   │   └── persistence/
│   │       └── postgres/        # Implémentations des repositories PostgreSQL
│   │
│   ├── application/
│   │   ├── availability/        # Logique métier des disponibilités
│   │   ├── team/                # Logique métier des équipes et de leurs réglages
│   │   └── guild/               # Langue et message d'accueil du serveur
│   │
│   ├── domain/
│   │   ├── availability/        # Semaines, créneaux, publication, fuseaux
│   │   ├── team/                # Modèle Team
│   │   └── guild/               # Modèle Guild (langue, accueil)
│   │
│   ├── i18n/                    # Textes du bot en français et en anglais
│   └── db/                      # Code généré par sqlc (ne pas modifier)
│
├── migrations/                  # Fichiers de migration SQL
├── Dockerfile               # Image du bot (distroless)
├── Dockerfile.migrate       # Image des migrations
├── .dockerignore            # Garde .env et .git hors du build
├── docker-compose.yml           # Stack locale dev
├── docker-compose.prod.yml      # Stack prod pour Coolify
├── sqlc.yaml
└── .env.example
```

## Schéma de base de données

```sql
-- Équipes d'un serveur et leurs réglages
CREATE TABLE teams (
    id              BIGSERIAL PRIMARY KEY,
    guild_id        TEXT NOT NULL,
    name            TEXT NOT NULL,  -- unique sur le serveur
    channel_id      TEXT,      -- Salon de publication
    role_id         TEXT,      -- Rôle à mentionner
    slot_mode       TEXT,      -- NULL/'halfday' = Après-midi/Soir, 'hourly' = par heure
    slot_start_hour SMALLINT,  -- mode hourly : première heure (0-23)
    slot_end_hour   SMALLINT,  -- mode hourly : fin du dernier créneau (24 = minuit, 25-29 = 1h-5h le lendemain)
    display_style   TEXT,      -- NULL/'classic' = embed texte, 'image' = grille en image
    language        TEXT,      -- NULL/'fr' = français, 'en' = anglais
    publish_weekday SMALLINT,  -- NULL = lundi · 0 = dimanche (semaine qui suit), 1 = lundi
    publish_hour    SMALLINT,  -- NULL = 10h · heure de publication (0-23, heure de l'équipe)
    active          BOOLEAN NOT NULL DEFAULT TRUE, -- FALSE = en pause (bot retiré, salon inaccessible)
    timezone        TEXT,      -- NULL = Europe/Paris · fuseau du pays ou décalage fixe ("UTC+2")
    member_timezones BOOLEAN NOT NULL DEFAULT FALSE, -- TRUE = international : chacun voit les créneaux à son heure
    full_at         SMALLINT,  -- NULL = couleurs relatives au meilleur créneau · sinon joueurs pour une case verte
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT teams_name_per_guild UNIQUE (guild_id, name)
);

-- Dernier message de la semaine publié par équipe
CREATE TABLE weekly_messages (
    team_id    BIGINT PRIMARY KEY REFERENCES teams (id) ON DELETE CASCADE,
    week       TEXT NOT NULL,  -- Clé de semaine ISO (ex: "2026-W10")
    channel_id TEXT NOT NULL,
    message_id TEXT NOT NULL
);

-- Disponibilités des membres
CREATE TABLE dispos (
    team_id    BIGINT   NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    user_id    TEXT     NOT NULL,
    day_index  SMALLINT NOT NULL,  -- 0=Lundi … 6=Dimanche
    slot_index SMALLINT NOT NULL,  -- Après-midi/Soir : 0, 1 · Par heure : 0 = première heure, 1 = +1h…
    week       TEXT     NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT dispos_unique_per_week UNIQUE (team_id, week, user_id, day_index, slot_index)
);

-- Membres « pas dispo cette semaine »
CREATE TABLE week_unavailable (
    team_id    BIGINT NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    week       TEXT NOT NULL,
    user_id    TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (team_id, week, user_id)
);

-- Ce qui concerne tout le serveur
CREATE TABLE guilds (
    guild_id           TEXT PRIMARY KEY,
    language           TEXT,  -- NULL = pas encore choisie · langue de « Mes équipes », de l'accueil et des nouvelles équipes
    welcome_channel_id TEXT,  -- message d'accueil posté à l'arrivée du bot, mis à jour pendant le setup
    welcome_message_id TEXT
);

-- Fuseau horaire choisi par chaque membre (mode Par heure)
CREATE TABLE user_timezone (
    user_id    TEXT PRIMARY KEY,
    timezone   TEXT NOT NULL,  -- fuseau du pays ("Europe/Paris") ou décalage fixe ("UTC+2")
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
```

## CI / CD

Le workflow GitHub Actions `.github/workflows/ci.yml` tourne sur chaque push sur `main` et chaque pull request :

| Job | Vérifie |
|---|---|
| **Lint and test** | `gofmt`, `go vet`, staticcheck, les tests avec le détecteur de courses (`-race`) sur un vrai PostgreSQL, et les migrations appliquées, annulées puis réappliquées |
| **Generated code up to date** | que le code sqlc correspond aux requêtes (`sqlc diff`) |
| **Docker images build** | que les images du bot et des migrations se construisent |
| **Deploy to Coolify** | sur `main` seulement, une fois tout le reste au vert : déclenche le déploiement Coolify |

Pour que seul du code testé parte en prod :

1. Dans Coolify, sur l'application : désactiver le **déploiement automatique** (il n'attend pas les tests), puis copier l'URL du **Deploy Webhook**.
2. Dans Coolify → **Keys & Tokens** : créer un **API token** avec le droit de déployer.
3. Dans GitHub → **Settings → Secrets and variables → Actions** : ajouter `COOLIFY_WEBHOOK` (l'URL) et `COOLIFY_TOKEN` (le token).
4. Dans GitHub → **Settings → Branches** : protéger `main` en exigeant que les jobs de la CI passent avant un merge.

Sans les deux secrets, le job de déploiement ne fait rien et le signale.

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
