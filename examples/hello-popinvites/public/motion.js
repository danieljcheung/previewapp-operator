document.addEventListener('DOMContentLoaded', () => {
  // 1. Check for reduced motion preference
  const prefersReducedMotion = window.matchMedia('(prefers-reduced-motion: reduce)').matches;

  // 2. Generate Decorative QR Grid inside the Ticket Pass
  const qr = document.querySelector('.pass-qr');
  if (qr) {
    for (let r = 0; r < 12; r++) {
      for (let c = 0; c < 12; c++) {
        const dot = document.createElement('div');
        dot.classList.add('qr-dot');
        
        // Top-left locator (3x3 square)
        if (r < 3 && c < 3) {
          dot.classList.add('corner');
        }
        // Top-right locator
        else if (r < 3 && c >= 9) {
          dot.classList.add('corner');
        }
        // Bottom-left locator
        else if (r >= 9 && c < 3) {
          dot.classList.add('corner');
        }
        // Mock pattern for the rest
        else {
          const rand = Math.random();
          if (rand > 0.6) {
            dot.classList.add('filled');
          } else if (rand > 0.45) {
            dot.classList.add('accent');
          }
        }
        qr.appendChild(dot);
      }
    }
  }

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
        this.vy = -(Math.random() * 0.4 + 0.1); // float upwards slowly
        this.size = Math.random() * 5 + 1.5;
        this.color = Math.random() > 0.5 ? '232, 216, 200' : '229, 193, 190'; // Champagne or Blush
        this.alpha = 0;
        this.maxAlpha = Math.random() * 0.5 + 0.1;
        this.life = 0;
        this.maxLife = Math.random() * 300 + 150;
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

  // 5. RSVP Interactions (Client-only state sync)
  const rsvpButtons = document.querySelectorAll('.btn-rsvp');
  const rsvpStateTexts = document.querySelectorAll('.rsvp-state');

  rsvpButtons.forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.preventDefault();
      
      rsvpButtons.forEach(b => {
        b.textContent = "You're on the list";
        b.disabled = true;
        b.style.background = 'rgba(152, 217, 194, 0.2)';
        b.style.color = '#98d9c2';
        b.style.borderColor = 'rgba(152, 217, 194, 0.4)';
        b.style.boxShadow = 'none';
      });

      rsvpStateTexts.forEach(txt => {
        txt.textContent = "RSVP held locally for this demo";
        txt.style.opacity = '1';
      });
    });
  });

  // 6. Calendar Event Generation & Download (.ics)
  const calendarButtons = document.querySelectorAll('.btn-calendar');
  calendarButtons.forEach(btn => {
    btn.addEventListener('click', (e) => {
      e.preventDefault();
      
      // EDT Toronto July 18, 2026 9:00 PM is UTC July 19, 2026 1:00 AM (01:00)
      // EDT Toronto July 19, 2026 1:00 AM is UTC July 19, 2026 5:00 AM (05:00)
      const icsLines = [
        'BEGIN:VCALENDAR',
        'VERSION:2.0',
        'PRODID:-//PopInvites//Lumen Bloom//EN',
        'BEGIN:VEVENT',
        'UID:lumen-bloom-2026-event@popinvites.com',
        'DTSTAMP:20260620T000000Z',
        'DTSTART:20260719T010000Z',
        'DTEND:20260719T050000Z',
        'SUMMARY:Lumen Bloom Soirée',
        'DESCRIPTION:An after-dark garden party and launch soirée. Dress code: moonlit formal.',
        'LOCATION:The Glasshouse\\, Toronto',
        'END:VEVENT',
        'END:VCALENDAR'
      ];
      
      const icsContent = icsLines.join('\r\n');
      const blob = new Blob([icsContent], { type: 'text/calendar;charset=utf-8;' });
      const url = URL.createObjectURL(blob);
      const link = document.createElement('a');
      link.href = url;
      link.download = 'lumen-bloom.ics';
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

  // 8. Countdown Timer to July 18 2026 21:00 America/Toronto (July 19 2026 01:00 UTC)
  const countdownTarget = new Date('2026-07-19T01:00:00Z').getTime();
  
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
