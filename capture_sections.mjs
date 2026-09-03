import { chromium } from 'playwright';

async function capture() {
  const browser = await chromium.launch();
  const page = await browser.newPage({ viewport: { width: 1440, height: 1100 } });
  
  await page.goto('http://localhost:8090', { waitUntil: 'networkidle' });
  await page.waitForTimeout(1000);

  // 1. Hero & Bento Cards (Top)
  await page.evaluate(() => window.scrollTo(0, 0));
  await page.waitForTimeout(400);
  await page.screenshot({ path: '/home/gio/projects/portfolio/screenshots/01-hero-and-bento.png' });

  // 2. Philosophy Pillars
  const philElem = await page.$('#philosophy');
  if (philElem) {
    await philElem.scrollIntoViewIfNeeded();
    await page.waitForTimeout(400);
    await page.screenshot({ path: '/home/gio/projects/portfolio/screenshots/02-philosophy-pillars.png' });
  }

  // 3. Expertise Drawer
  const expElem = await page.$('#expertise');
  if (expElem) {
    await expElem.scrollIntoViewIfNeeded();
    await page.waitForTimeout(400);
    await page.screenshot({ path: '/home/gio/projects/portfolio/screenshots/03-expertise-drawer.png' });
  }

  // 4. Case Studies
  const projElem = await page.$('#projects');
  if (projElem) {
    await projElem.scrollIntoViewIfNeeded();
    await page.waitForTimeout(400);
    await page.screenshot({ path: '/home/gio/projects/portfolio/screenshots/04-case-studies.png' });
  }

  // 5. Contact Section
  const contactElem = await page.$('#contact');
  if (contactElem) {
    await contactElem.scrollIntoViewIfNeeded();
    await page.waitForTimeout(400);
    await page.screenshot({ path: '/home/gio/projects/portfolio/screenshots/05-contact-form.png' });
  }

  await browser.close();
  console.log('All 5 section screenshots captured successfully');
}

capture().catch(console.error);
