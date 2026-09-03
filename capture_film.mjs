import { chromium } from 'playwright';

async function captureFilm() {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1440, height: 1200 } });
  
  await page.goto('http://localhost:8090', { waitUntil: 'networkidle' });
  await page.waitForTimeout(1500);

  // Scroll to projects section
  const proj = await page.$('#projects');
  if (proj) {
    await proj.scrollIntoViewIfNeeded();
    await page.waitForTimeout(1000);
    await page.screenshot({ path: '/home/gio/projects/portfolio/screenshots/threejs-film-accordion.png' });
  }

  await browser.close();
  console.log('Film accordion screenshot captured');
}

captureFilm().catch(console.error);
