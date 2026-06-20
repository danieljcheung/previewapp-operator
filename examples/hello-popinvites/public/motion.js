document.addEventListener('DOMContentLoaded', () => {
  // 1. Check for reduced motion preference
  const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  // 3. Canvas Atmosphere Particle Animation
  const canvas = document.getElementById('atmosphere');
  if (canvas) {
    const ctx = canvas.getContext('2d');
    let animationFrameId;
    let particles = [];

    const resizeCanvas = () => {
      canvas.width = window.innerWidth;
      canvas.height = window.innerHeight;
    };
    window.addEventListener('resize', resizeCanvas);
    resizeCanvas();

    class Particle {
      constructor() {
        this.reset();
        // Distribute initial particles across the screen
        this.y = Math.random() * canvas.height;
      }
      reset() {
        this.x = Math.random() * canvas.width;
        this.y = canvas.height + Math.random() * 20; // start slightly below screen
        this.vx = (Math.random() - 0.5) * 0.3;
        this.vy = -(Math.random() * 0.5 + 0.2); // float upwards slowly (like boba pearls)
        this.size = Math.random() * 8 + 3; // Boba pearls are slightly larger and softer
        const colors = [
          '232, 216, 200', // Champagne / Milk tea
          '152, 217, 194', // Mint
          '255, 255, 255', // Pearl White
          '210, 180, 140'  // Soft Oolong Milk tea
        ];
        this.color = colors[Math.floor(Math.random() * colors.length)];
        this.alpha = 0;
        this.maxAlpha = Math.random() * 0.4 + 0.1;
        this.life = 0;
        this.maxLife = Math.random() * 400 + 200;
      }
      update() {
        this.x += this.vx;
        this.y += this.vy;
        this.life++;

        // Fade in at start, fade out at end
        if (this.life < 50) {
          this.alpha = (this.life / 50) * this.maxAlpha;
        } else if (this.life > this.maxLife - 50) {
          this.alpha = ((this.maxLife - this.life) / 50) * this.maxAlpha;
        } else {
          this.alpha = this.maxAlpha;
        }

        if (this.y < -20 || this.alpha <= 0 || this.life >= this.maxLife) {
          this.reset();
        }
      }
      draw() {
        ctx.beginPath();
        const gradient = ctx.createRadialGradient(this.x, this.y, 0, this.x, this.y, this.size);
        gradient.addColorStop(0, `rgba(${this.color}, ${this.alpha})`);
        gradient.addColorStop(1, `rgba(${this.color}, 0)`);
        ctx.fillStyle = gradient;
        ctx.arc(this.x, this.y, this.size, 0, Math.PI * 2);
        ctx.fill();
      }
    }

    if (!prefersReducedMotion) {
      // Initialize particles
      const particleCount = 35;
      for (let i = 0; i < particleCount; i++) {
        particles.push(new Particle());
      }

      const animate = () => {
        ctx.clearRect(0, 0, canvas.width, canvas.height);
        particles.forEach(p => {
          p.update();
          p.draw();
        });
        animationFrameId = requestAnimationFrame(animate);
      };
      animate();
    }
  }

  // 4. Pointer-based Parallax Effect
  if (!prefersReducedMotion) {
    window.addEventListener('mousemove', (e) => {
      // Normalized coordinates from -0.5 to 0.5
      const pointerX = (e.clientX / window.innerWidth) - 0.5;
      const pointerY = (e.clientY / window.innerHeight) - 0.5;
      
      document.documentElement.style.setProperty('--pointer-x', pointerX);
      document.documentElement.style.setProperty('--pointer-y', pointerY);
    });
  }

  // 5. Share Interactions (Client-only state sync)
  const shareButtons = document.querySelectorAll('.btn-share-team');
  const shareStateTexts = document.querySelectorAll('.share-state');

  shareButtons.forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.preventDefault();
      
      // Copy link to clipboard
      const inviteUrl = "https://hello.popinvites.com";
      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(inviteUrl).catch(err => console.error(err));
      }

      shareButtons.forEach(b => {
        b.textContent = "Marked as shared";
        b.disabled = true;
        b.style.background = 'rgba(152, 217, 194, 0.2)';
        b.style.color = '#98d9c2';
        b.style.borderColor = 'rgba(152, 217, 194, 0.4)';
        b.style.boxShadow = 'none';
      });

      shareStateTexts.forEach(txt => {
        txt.textContent = "Share note saved locally for this demo";
        txt.style.opacity = '1';
      });

      // Also trigger toast for clipboard copy
      if (toast) {
        toast.textContent = "Link copied! Share it on Slack or Teams.";
        toast.classList.add('show');
        setTimeout(() => {
          toast.classList.remove('show');
        }, 2500);
      }
    });
  });

  // 6. Menu jump
  const menuButtons = document.querySelectorAll('.btn-menu');
  menuButtons.forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.preventDefault();
      const menu = document.getElementById('menu');
      if (!menu) return;
      const targetY = menu.getBoundingClientRect().top + window.scrollY;
      document.scrollingElement.scrollTop = targetY;
      document.documentElement.scrollTop = targetY;
      window.scrollTo({ top: targetY, behavior: prefersReducedMotion ? 'auto' : 'smooth' });
    });
  });

  // 6. Calendar Event Generation & Download (.ics)
  const calendarButtons = document.querySelectorAll('.btn-calendar');
  calendarButtons.forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.preventDefault();
      
      // EDT Toronto July 16, 2026 2:00 PM is UTC July 16, 2026 6:00 PM (18:00)
      // EDT Toronto July 16, 2026 4:00 PM is UTC July 16, 2026 8:00 PM (20:00)
      const icsLines = [
        'BEGIN:VCALENDAR',
        'VERSION:2.0',
        'PRODID:-//PopInvites//PopUp Pearl//EN',
        'BEGIN:VEVENT',
        'UID:popup-pearl-2026-event@popinvites.com',
        'DTSTAMP:20260620T000000Z',
        'DTSTART:20260716T180000Z',
        'DTEND:20260716T200000Z',
        'SUMMARY:PopUp Pearl Bubble Tea Social',
        'DESCRIPTION:Internal corporate bubble tea social and team break. Bring your team!',
        'LOCATION:HQ 4th Floor Garden Terrace',
        'END:VEVENT',
        'END:VCALENDAR'
      ];
      
      const icsContent = icsLines.join('\r\n');
      const blob = new Blob([icsContent], { type: 'text/calendar;charset=utf-8;' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = 'popup-pearl.ics';
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    });
  });

  // 7. Copy Invite Link Functionality
  const copyButtons = document.querySelectorAll('.btn-copy');
  const toast = document.getElementById('toast');

  copyButtons.forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.preventDefault();
      const inviteUrl = "https://hello.popinvites.com";

      const showToast = () => {
        if (toast) {
          toast.textContent = "Link copied to clipboard";
          toast.classList.add('show');
          setTimeout(() => {
            toast.classList.remove('show');
          }, 2500);
        }
      };

      const fallbackCopy = () => {
        const input = document.createElement('input');
        input.value = inviteUrl;
        document.body.appendChild(input);
        input.select();
        try {
          document.execCommand('copy');
          showToast();
        } catch (err) {
          console.error('Copy fallback failed', err);
        }
        document.body.removeChild(input);
      };

      if (navigator.clipboard && navigator.clipboard.writeText) {
        navigator.clipboard.writeText(inviteUrl)
          .then(showToast)
          .catch(fallbackCopy);
      } else {
        fallbackCopy();
      }
    });
  });

  // 8. Countdown Timer to July 16 2026 14:00 America/Toronto (July 16 2026 18:00 UTC)
  const countdownTarget = new Date('2026-07-16T18:00:00Z').getTime();
  
  const dVal = document.querySelector('.days-val');
  const hVal = document.querySelector('.hours-val');
  const mVal = document.querySelector('.minutes-val');
  const sVal = document.querySelector('.seconds-val');
  const countdownContainer = document.querySelector('[data-countdown]');

  const updateCountdown = () => {
    const now = new Date().getTime();
    const diff = countdownTarget - now;

    if (diff <= 0) {
      if (countdownContainer) {
        countdownContainer.innerHTML = '<div class="serif-font" style="font-size: 24px; color: var(--color-champagne);">The event has begun</div>';
      }
      return;
    }

    const days = Math.floor(diff / (1000 * 60 * 60 * 24));
    const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60));
    const minutes = Math.floor((diff % (1000 * 60 * 60)) / (1000 * 60));
    const seconds = Math.floor((diff % (1000 * 60)) / 1000);

    if (dVal && hVal && mVal && sVal) {
      dVal.textContent = String(days).padStart(2, '0');
      hVal.textContent = String(hours).padStart(2, '0');
      mVal.textContent = String(minutes).padStart(2, '0');
      sVal.textContent = String(seconds).padStart(2, '0');
    }
  };

  updateCountdown();
  setInterval(updateCountdown, 1000);
});
