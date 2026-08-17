const header = document.querySelector("[data-header]");
const updateHeader = () => header?.classList.toggle("is-scrolled", window.scrollY > 24);

updateHeader();
window.addEventListener("scroll", updateHeader, { passive: true });

const gallery = document.querySelector("[data-gallery]");

if (gallery) {
  const slides = [...gallery.querySelectorAll(".gallery-card")];
  const previousButton = gallery.querySelector("[data-gallery-previous]");
  const nextButton = gallery.querySelector("[data-gallery-next]");
  const currentLabel = gallery.querySelector("[data-gallery-current]");
  const totalLabel = gallery.querySelector("[data-gallery-total]");
  let currentSlide = 0;

  const preloadSlide = (slideIndex) => {
    if (slides.length === 0) return;
    const normalizedIndex = (slideIndex + slides.length) % slides.length;
    const image = slides[normalizedIndex].querySelector("img");
    if (image) image.loading = "eager";
  };

  const showGallerySlide = (slideIndex) => {
    if (slides.length === 0) return;
    currentSlide = (slideIndex + slides.length) % slides.length;

    slides.forEach((slide, index) => {
      const isCurrent = index === currentSlide;
      slide.hidden = !isCurrent;
      slide.classList.toggle("is-active", isCurrent);
      if (isCurrent) slide.classList.add("is-visible");
      slide.setAttribute("aria-hidden", String(!isCurrent));
    });

    preloadSlide(currentSlide);
    preloadSlide(currentSlide + 1);

    if (currentLabel) currentLabel.textContent = String(currentSlide + 1);
    if (totalLabel) totalLabel.textContent = String(slides.length);
    if (previousButton) previousButton.disabled = slides.length < 2;
    if (nextButton) nextButton.disabled = slides.length < 2;
  };

  previousButton?.addEventListener("click", () => showGallerySlide(currentSlide - 1));
  nextButton?.addEventListener("click", () => showGallerySlide(currentSlide + 1));
  gallery.addEventListener("keydown", (event) => {
    if (event.key === "ArrowLeft") {
      event.preventDefault();
      showGallerySlide(currentSlide - 1);
    }
    if (event.key === "ArrowRight") {
      event.preventDefault();
      showGallerySlide(currentSlide + 1);
    }
  });
  showGallerySlide(0);
}

const revealObserver = new IntersectionObserver(
  (entries, observer) => {
    for (const entry of entries) {
      if (!entry.isIntersecting) continue;
      entry.target.classList.add("is-visible");
      observer.unobserve(entry.target);
    }
  },
  { rootMargin: "0px 0px -8%", threshold: 0.08 },
);

document.querySelectorAll(".reveal").forEach((element) => revealObserver.observe(element));

const lightbox = document.querySelector("[data-lightbox-dialog]");
const lightboxImage = lightbox?.querySelector("[data-lightbox-image]");
const lightboxCaption = lightbox?.querySelector("[data-lightbox-caption]");

if (lightbox && lightboxImage && lightboxCaption && typeof lightbox.showModal === "function") {
  document.querySelectorAll("[data-lightbox]").forEach((link) => {
    link.addEventListener("click", (event) => {
      event.preventDefault();

      const preview = link.querySelector("img");
      lightboxImage.src = link.href;
      lightboxImage.alt = preview?.alt || "Goro gameplay screenshot";
      lightboxCaption.textContent = link.dataset.caption || "Goro in game";
      lightbox.showModal();
    });
  });

  lightbox.querySelector("[data-lightbox-close]")?.addEventListener("click", () => lightbox.close());
  lightbox.addEventListener("click", (event) => {
    if (event.target === lightbox) lightbox.close();
  });
}

const platformName = navigator.userAgentData?.platform || navigator.platform || navigator.userAgent;
const normalizedPlatform = platformName.toLowerCase();

let platform;
let primaryDownload;

if (normalizedPlatform.includes("win")) {
  platform = "windows";
  primaryDownload = {
    label: "Download for Windows",
    url: "https://github.com/kivutar/goro/releases/latest/download/goro-windows-x86_64.exe",
  };
} else if (normalizedPlatform.includes("mac")) {
  platform = "macos";
  primaryDownload = {
    label: "Download for macOS",
    url: "#download",
  };
} else if (normalizedPlatform.includes("linux")) {
  platform = "linux";
  primaryDownload = {
    label: "Download for Linux",
    url: "https://github.com/kivutar/goro/releases/latest/download/goro-linux-x86_64",
  };
}

if (platform && primaryDownload) {
  const primaryButton = document.querySelector("[data-primary-download]");
  const primaryLabel = document.querySelector("[data-primary-label]");

  if (primaryButton) primaryButton.href = primaryDownload.url;
  if (primaryLabel) primaryLabel.textContent = primaryDownload.label;
  document.querySelector(`[data-platform-card="${platform}"]`)?.classList.add("is-recommended");
}

const year = document.querySelector("[data-year]");
if (year) year.textContent = new Date().getFullYear().toString();
