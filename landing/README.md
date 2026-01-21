# Chorus Landing Page

Beautiful, responsive marketing website for Chorus messaging app.

## Pages

- **Home** (`index.html`) - Main landing page with hero, features, and call-to-action
- **Download** (`download.html`) - Platform download options and requirements  
- **About** (`about.html`) - Company mission, values, and story

## Running Locally

### Option 1: Simple HTTP Server (Node.js)

```bash
cd landing
node server.js
```

Then open http://localhost:5000

### Option 2: Python HTTP Server

```bash
cd landing
python -m http.server 5000
```

Then open http://localhost:5000

### Option 3: Open Directly

Simply open `index.html` in your web browser.

## Features

- ✅ Fully responsive design (mobile, tablet, desktop)
- ✅ Modern gradient backgrounds and animations
- ✅ Smooth scrolling and transitions
- ✅ Interactive phone mockup with chat preview
- ✅ Language showcase section
- ✅ Feature cards with hover effects
- ✅ Mobile-friendly navigation menu
- ✅ Professional footer with links
- ✅ No external dependencies (except Google Fonts)

## Technologies

- HTML5
- CSS3 (Flexbox, Grid, Animations)
- Vanilla JavaScript
- Google Fonts (Inter)

## Customization

All colors and styles are defined in CSS custom properties at the top of `styles.css`:

```css
:root {
    --primary: #667eea;
    --secondary: #764ba2;
    --accent: #f093fb;
    /* ... */
}
```

## Browser Support

- Chrome 90+
- Firefox 88+
- Safari 14+
- Edge 90+
