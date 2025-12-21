# Build Instructions for Fluxor.io Website

## Prerequisites

1. **Node.js** (version 18 or higher)
   - Download from: https://nodejs.org/
   - Verify installation: `node --version` and `npm --version`

## Build Steps

### 1. Install Dependencies

```bash
cd web
npm install
```

This will install all required packages:
- React 19.2.3
- React DOM 19.2.3
- Vite 6.2.0
- TypeScript 5.8.2
- Lucide React (icons)
- @vitejs/plugin-react

### 2. Development Server

To run the development server with hot reload:

```bash
npm run dev
```

The website will be available at: http://localhost:3000

### 3. Production Build

To create an optimized production build:

```bash
npm run build
```

This will:
- Compile TypeScript to JavaScript
- Bundle React components
- Optimize assets
- Generate static files in `web/dist/` directory

### 4. Preview Production Build

To preview the production build locally:

```bash
npm run preview
```

## Build Output

After running `npm run build`, the following files will be generated in `web/dist/`:

- `index.html` - Main HTML file
- `assets/` - Bundled JavaScript and CSS files
- Other static assets

## Deployment

The `web/dist/` directory contains all static files needed for deployment. You can:

1. **Deploy to static hosting:**
   - Netlify
   - Vercel
   - GitHub Pages
   - AWS S3 + CloudFront
   - Any static file server

2. **Serve with Go server:**
   - Use `pkg/web/fasthttp_server.go` to serve static files
   - Point to `web/dist/` directory

## Troubleshooting

### Node.js not found
- Install Node.js from https://nodejs.org/
- Restart terminal after installation
- Verify with `node --version`

### npm install fails
- Clear npm cache: `npm cache clean --force`
- Delete `node_modules` and `package-lock.json`
- Run `npm install` again

### Build errors
- Check TypeScript errors: `npx tsc --noEmit`
- Verify all imports are correct
- Ensure all dependencies are installed

## Project Structure

```
web/
├── components/          # React components
│   ├── Navbar.tsx
│   ├── Hero.tsx
│   ├── Features.tsx
│   ├── Architecture.tsx
│   ├── Comparison.tsx
│   └── Footer.tsx
├── index.html          # HTML entry point
├── index.tsx           # React entry point
├── App.tsx             # Main app component
├── package.json        # Dependencies and scripts
├── tsconfig.json       # TypeScript configuration
├── vite.config.ts      # Vite build configuration
└── dist/               # Build output (generated)
```

