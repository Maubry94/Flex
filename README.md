# Flex

Flex est un serveur multimédia personnel, auto-hébergé et conçu pour parcourir puis lire ses propres vidéos depuis un navigateur récent.

Le projet est au début de son développement. Il permet déjà d'ajouter des bibliothèques, d'indexer les vidéos avec FFmpeg, de générer leurs miniatures, de les lire directement ou après conversion HLS en H.264, et de reprendre automatiquement une lecture interrompue. Flex prend également en charge plusieurs comptes, les favoris, les collections et les progressions de lecture propres à chaque utilisateur.

La conversion HLS est actuellement réalisée entièrement avant la première lecture puis conservée dans le cache. Le transcodage progressif, les limites de cache et l'accélération matérielle arriveront dans des versions suivantes.

## Démarrage avec Docker Compose

Prérequis : Docker avec le plugin Compose.

```bash
cp .env.example .env
mkdir -p data/config data/cache media
docker compose up --build
```

Flex est ensuite disponible sur <http://localhost:8080>.

Lors de la première ouverture, Flex demande de créer le compte administrateur. Cet administrateur peut ensuite créer et gérer les autres comptes depuis le menu du profil, dans **Administration → Utilisateurs**.

Placez quelques vidéos dans `./media`, ou modifiez `FLEX_MEDIA_PATH` dans `.env`. Les médias sont toujours montés en lecture seule dans le conteneur.

Pour utiliser une image publiée sans construire le projet localement :

```bash
docker compose pull
docker compose up -d --no-build
```

## Développement

Une seule commande démarre Vite sur le port 5173 et le serveur Go sur le port 8080 :

```bash
npm run dev
```

L'interface est disponible sur <http://localhost:5173>. Vite transmet automatiquement les appels `/api` au serveur Go.

Pour démarrer exceptionnellement l'interface sans le serveur :

```bash
npm run dev:web
```

Sans Docker, utilisez Node.js 24 et Go 1.25. Démarrez d'abord l'interface :

```bash
npm install
npm run dev:web
```

Puis, dans un autre terminal :

```bash
cd apps/server
FLEX_WEB_DIR=../web/dist go run ./cmd/flex
```

## TrueNAS SCALE

À partir de TrueNAS SCALE 24.10, ouvrez **Apps → Discover Apps → Install via YAML**, puis utilisez le contenu de `compose.yml` en remplaçant les chemins par des datasets TrueNAS, par exemple :

```yaml
volumes:
  - /mnt/tank/apps/flex/config:/config
  - /mnt/tank/apps/flex/cache:/cache
  - /mnt/tank/media/videos:/media:ro
```

Pour TrueNAS, supprimez la section `build` du YAML et utilisez une image Flex versionnée plutôt que `latest`. Vérifiez également que l'UID/GID choisi dispose d'un accès en lecture aux médias et en écriture aux datasets `config` et `cache`.

## Configuration

| Variable | Valeur par défaut | Description |
| --- | --- | --- |
| `FLEX_PORT` | `8080` | Port publié sur l'hôte par Compose. |
| `FLEX_UID` / `FLEX_GID` | `1000` | Identité Linux du processus dans le conteneur. |
| `FLEX_CONFIG_PATH` | `./data/config` | Configuration persistante et base SQLite. |
| `FLEX_CACHE_PATH` | `./data/cache` | Miniatures et transcodages temporaires. |
| `FLEX_MEDIA_PATH` | `./media` | Racine des vidéos, montée en lecture seule. |
| `FLEX_IMAGE` | `ghcr.io/flex-media/flex:latest` | Image utilisée par Compose. |

Flex protège l'interface avec des comptes locaux et des sessions HTTP sécurisées. Pour une exposition sur Internet, utilisez néanmoins HTTPS et conservez une protection périmétrique telle que Cloudflare Access devant l'application. Flex ne configure ni TLS, ni tunnel, ni limitation réseau à votre place.

Les bibliothèques, leur analyse et les métadonnées éditoriales sont administrées par les comptes administrateur. Les favoris, la progression de lecture et les collections sont personnels à chaque utilisateur.

## Qualité

```bash
npm run lint
npm run typecheck
npm test
npm run build

cd apps/server
go test ./...
go vet ./...
```

TypeScript est configuré en mode strict et ESLint interdit explicitement `any`. Les réponses provenant de l'API sont traitées comme `unknown` puis validées à la frontière du frontend.
