/**
 * Rizzer Web App - Core Logic & Data Engine
 * Powered by Web Audio API, Web Speech API, HTML5 Canvas & LocalStorage
 */

// ================= RIZZ LINES DATABASE =================
const RIZZ_DATABASE = [
  // --- 🔥 SMOOTH & FLIRTY ---
  { id: 101, text: "Có 365 ngày trong năm, nhưng ngày nào tui cũng chỉ nghĩ về ní thôi.", cat: "smooth", score: 96 },
  { id: 102, text: "Cậu có biết sự khác biệt giữa cậu và ngôi sao là gì không? Ngôi sao ở trên trời, còn cậu ở trong tim tui.", cat: "smooth", score: 98 },
  { id: 103, text: "Tui không phải nhiếp ảnh gia, nhưng tui có thể tưởng tượng bức tranh tương lai của hai đứa mình.", cat: "smooth", score: 94 },
  { id: 104, text: "Mặt trời mọc ở đằng Đông, còn nụ cười của cậu thì làm tim tui rung động.", cat: "smooth", score: 95 },
  { id: 105, text: "Tay cậu trông hơi nặng đấy... Để tui nắm hộ cho nhẹ bớt nha!", cat: "smooth", score: 99 },
  { id: 106, text: "Người ta nói tình yêu cần thời gian, nhưng tui chỉ cần 1 giây nhìn cậu là đã đổ rồi.", cat: "smooth", score: 92 },
  { id: 107, text: "Cậu ơi, dạo này cậu có thấy mỏi chân không? Vì cậu cứ chạy trong đầu tui cả ngày đấy!", cat: "smooth", score: 97 },
  { id: 108, text: "Nếu có 1 ước nguyện ngay lúc này, tui chỉ ước nụ cười đó là dành riêng cho tui.", cat: "smooth", score: 93 },
  { id: 109, text: "Mắt cậu đẹp thật đấy, nhưng tui thích đôi mắt ấy hơn khi nó nhìn về phía tui.", cat: "smooth", score: 99 },
  { id: 110, text: "Cậu có muốn làm người nắm giữ chìa khóa trái tim tui không? Miễn phí tiền thuê luôn!", cat: "smooth", score: 91 },

  // --- 🧠 BRAINROT & SIGMA ---
  { id: 201, text: "Are you Fanum? Because you just taxed 100% of my thoughts fr fr.", cat: "brainrot", score: 99 },
  { id: 202, text: "Em có phải Skibidi Toilet không? Vì em làm tim anh xả hết lo âu, No Cap!", cat: "brainrot", score: 97 },
  { id: 203, text: "Ní có mewing không mà sao góc nghiêng thần thánh làm tim tui đứng hình vậy?", cat: "brainrot", score: 95 },
  { id: 204, text: "Rizz này không phải Rizz thường, đây là Rizz Level 9999 Sigma Male Kai Cenat approved!", cat: "brainrot", score: 100 },
  { id: 205, text: "Are you Kai Cenat? Cause you just brought maximum Gyatt energy into my life!", cat: "brainrot", score: 96 },
  { id: 206, text: "Bỏ qua Livestream đi, hôm nay tui chỉ muốn Stream nụ cười của ní 24/7 thui.", cat: "brainrot", score: 94 },
  { id: 207, text: "Tình cảm này là Pure Rizz, không scam, không mid, dính hơn cả keo 502!", cat: "brainrot", score: 93 },
  { id: 208, text: "Mọi người bảo tui bị delulu, nhưng delulu vì ní thì vạn lần valid!", cat: "brainrot", score: 98 },
  { id: 209, text: "Ní chính là Main Character trong cuốn sách cuộc đời tui, NPC khác không có cửa!", cat: "brainrot", score: 96 },
  { id: 210, text: "Bảo vệ tim tui khỏi ní còn khó hơn bảo vệ Win Streak 100 trận!", cat: "brainrot", score: 92 },

  // --- 💻 DEV & TECH BRO ---
  { id: 301, text: "Em có phải là bug không? Vì trong đầu anh lúc nào cũng chỉ có em, debug hoài không hết.", cat: "tech", score: 98 },
  { id: 302, text: "Ní là main branch hay sao mà tui chỉ muốn merge cuộc đời tui vào ní?", cat: "tech", score: 96 },
  { id: 303, text: "System.out.println('I love you'); - Không có try-catch nào bắt được tình cảm này đâu!", cat: "tech", score: 95 },
  { id: 304, text: "Tình cảm tui dành cho ní như vòng lặp while(true), không bao giờ break!", cat: "tech", score: 97 },
  { id: 305, text: "Anh sẵn sàng làm API Endpoint để em gửi request yêu thương 24/7.", cat: "tech", score: 99 },
  { id: 306, text: "Em là CSS custom property hay sao mà overwrite hết mọi chuẩn mực trong tim anh?", cat: "tech", score: 94 },
  { id: 307, text: "Cần gì Docker container khi tim anh đã đóng gói trọn vẹn hình bóng em?", cat: "tech", score: 93 },
  { id: 308, text: "Cho anh xin SSH key vào trái tim em được không? Anh hứa mã hóa 256-bit an toàn tuyệt đối!", cat: "tech", score: 96 },
  { id: 309, text: "Tình yêu của anh dành cho em như RAM vậy, lúc nào cũng Full Usage!", cat: "tech", score: 91 },
  { id: 310, text: "Em là Fiber router hay sao mà kết nối phát là tim anh chạy với tốc độ ánh sáng?", cat: "tech", score: 92 },

  // --- ☕ VIETNAMESE LOCAL RIZZ ---
  { id: 401, text: "Dạo này Hà Nội/Sài Gòn bão bụi quá, nhưng không bụi bằng sự chú ý anh dành cho em.", cat: "vietnam", score: 95 },
  { id: 402, text: "Muốn ăn bánh mì phải có patê, muốn tim không trống phải mê em nè.", cat: "vietnam", score: 96 },
  { id: 403, text: "Thịt mỡ dưa hành câu đối đỏ, thì thầm nói nhỏ 'Yêu em dữ nè'.", cat: "vietnam", score: 98 },
  { id: 404, text: "Trà sữa phải có trân châu, thanh xuân này phải có nhau mới bền.", cat: "vietnam", score: 94 },
  { id: 405, text: "Anh đây chẳng thích rượu trà, chỉ thích chiều chiều qua nhà đón em.", cat: "vietnam", score: 97 },
  { id: 406, text: "Hà Nội đưa đón gió mùa, còn anh đưa đón em về làm dâu.", cat: "vietnam", score: 99 },
  { id: 407, text: "Nắng Sài Gòn làm anh say nắng, nụ cười của em làm anh say đắm.", cat: "vietnam", score: 93 },
  { id: 408, text: "Nhà em có bán rượu không? Sao nói chuyện với em làm anh say quá!", cat: "vietnam", score: 91 },
  { id: 409, text: "Vector chỉ có một chiều, anh đây cũng chỉ một lòng yêu em.", cat: "vietnam", score: 92 },
  { id: 410, text: "Em ơi chớ uống trà đào, nhìn em một cái ngủ nào cũng mơ.", cat: "vietnam", score: 90 },

  // --- 🌸 CUTE & WHOLESOME ---
  { id: 501, text: "Cậu ơi, cho tui mượn la bàn với, tui lỡ lạc vào ánh mắt cậu rồi.", cat: "cute", score: 94 },
  { id: 502, text: "Tim tui bé lắm, chỉ chứa đủ nụ cười của cậu thôi đó.", cat: "cute", score: 96 },
  { id: 503, text: "Hôm nay trời đẹp thế này, không hẹn hò với tui là phí lắm đó nha!", cat: "cute", score: 93 },
  { id: 504, text: "Có một con mèo nhỏ cứ kêu 'meow meow', còn tim tui cứ kêu 'yêu cậu yêu cậu'.", cat: "cute", score: 97 },
  { id: 505, text: "Cậu thích đồ ngọt không? Vì tui thấy cậu ngọt ngào nhất trên đời rồi!", cat: "cute", score: 95 },
  { id: 506, text: "Tui định tặng cậu một món quà, nhưng hộp quà nào cũng nhỏ hơn tình cảm của tui.", cat: "cute", score: 92 },
  { id: 507, text: "Trời nổ giông bão ngoài kia, nhưng trong tim tui bình yên khi có cậu.", cat: "cute", score: 98 },
  { id: 508, text: "Mong ước nhỏ nhoi mỗi sáng: Mở mắt ra là thấy tin nhắn chúc ngủ ngon của cậu.", cat: "cute", score: 91 },

  // --- ⚡ UNHINGED & BOLD ---
  { id: 601, text: "Xin lỗi ní, ní có phải bảo hiểm xã hội không? Vì tui muốn gắn bó cả đời!", cat: "unhinged", score: 96 },
  { id: 602, text: "Em có tin vào tình yêu từ cái nhìn đầu tiên không, hay để anh đi qua lần nữa?", cat: "unhinged", score: 98 },
  { id: 603, text: "Nếu yêu em là phạm tội, anh nguyện nhận án chung thân không ân xá!", cat: "unhinged", score: 97 },
  { id: 604, text: "Mẹ anh bảo người đẹp hay gây phiền phức, và em chính là rắc rối lớn nhất đời anh.", cat: "unhinged", score: 99 },
  { id: 605, text: "Bác sĩ bảo anh thiếu vitamin U (You) trầm trọng, em kê đơn cứu anh đi!", cat: "unhinged", score: 94 },
  { id: 606, text: "Tay em có mỏi không? Nếu mỏi thì để anh gánh cả thế giới hộ em luôn!", cat: "unhinged", score: 95 }
];

// ================= APP STATE =================
let currentCategory = 'all';
let currentLine = null;
let favoriteIds = JSON.parse(localStorage.getItem('rizzer_favs') || '[]');
let isAudioMuted = false;
let canvasTheme = 'neon';

// Chat Simulator State
let currentBot = 'crush_cute';
let chatRizzScore = 40;
let chatHistory = [];

// ================= WEB AUDIO SYNTHESIZER =================
const audioCtx = new (window.AudioContext || window.webkitAudioContext)();

function playSound(type) {
  if (isAudioMuted) return;
  try {
    if (audioCtx.state === 'suspended') {
      audioCtx.resume();
    }
    const osc = audioCtx.createOscillator();
    const gain = audioCtx.createGain();
    osc.connect(gain);
    gain.connect(audioCtx.destination);

    const now = audioCtx.currentTime;

    if (type === 'click') {
      osc.type = 'sine';
      osc.frequency.setValueAtTime(440, now);
      osc.frequency.exponentialRampToValueAtTime(880, now + 0.08);
      gain.gain.setValueAtTime(0.15, now);
      gain.gain.exponentialRampToValueAtTime(0.01, now + 0.08);
      osc.start(now);
      osc.stop(now + 0.08);
    } else if (type === 'magic') {
      osc.type = 'triangle';
      osc.frequency.setValueAtTime(523.25, now); // C5
      osc.frequency.setValueAtTime(659.25, now + 0.08); // E5
      osc.frequency.setValueAtTime(783.99, now + 0.16); // G5
      osc.frequency.setValueAtTime(1046.50, now + 0.24); // C6
      gain.gain.setValueAtTime(0.2, now);
      gain.gain.exponentialRampToValueAtTime(0.01, now + 0.35);
      osc.start(now);
      osc.stop(now + 0.35);
    } else if (type === 'success') {
      osc.type = 'sine';
      osc.frequency.setValueAtTime(587.33, now);
      osc.frequency.exponentialRampToValueAtTime(1174.66, now + 0.2);
      gain.gain.setValueAtTime(0.25, now);
      gain.gain.exponentialRampToValueAtTime(0.01, now + 0.25);
      osc.start(now);
      osc.stop(now + 0.25);
    } else if (type === 'copy') {
      osc.type = 'square';
      osc.frequency.setValueAtTime(800, now);
      osc.frequency.exponentialRampToValueAtTime(1200, now + 0.06);
      gain.gain.setValueAtTime(0.1, now);
      gain.gain.exponentialRampToValueAtTime(0.01, now + 0.06);
      osc.start(now);
      osc.stop(now + 0.06);
    }
  } catch (e) {
    console.log('Audio playback error:', e);
  }
}

function toggleAudio() {
  isAudioMuted = !isAudioMuted;
  document.getElementById('audio-icon-on').classList.toggle('hidden', isAudioMuted);
  document.getElementById('audio-icon-off').classList.toggle('hidden', !isAudioMuted);
  showToast(isAudioMuted ? 'Muted sound effects 🔇' : 'Sound effects enabled 🔊');
}

// ================= NAVIGATION LOGIC =================
function switchTab(tabName) {
  playSound('click');
  document.querySelectorAll('.tab-content').forEach(el => el.classList.add('hidden'));
  document.querySelectorAll('.nav-btn').forEach(el => el.classList.remove('active'));

  const targetTab = document.getElementById(`tab-${tabName}`);
  const targetNav = document.getElementById(`nav-${tabName}`);

  if (targetTab) targetTab.classList.remove('hidden');
  if (targetNav) targetNav.classList.add('active');

  if (tabName === 'favorites') {
    renderFavorites();
  }
}

// ================= RIZZ GENERATOR ENGINE =================
function selectCategory(cat) {
  playSound('click');
  currentCategory = cat;
  document.querySelectorAll('.cat-pill').forEach(btn => {
    btn.classList.remove('active');
  });

  const activeBtn = event.currentTarget;
  if (activeBtn) activeBtn.classList.add('active');

  generateRizz();
}

function getFilteredLines() {
  if (currentCategory === 'all') return RIZZ_DATABASE;
  return RIZZ_DATABASE.filter(line => line.cat === currentCategory);
}

function generateRizz() {
  playSound('magic');
  const pool = getFilteredLines();
  if (pool.length === 0) return;

  let nextLine = pool[Math.floor(Math.random() * pool.length)];
  // avoid same line twice if possible
  if (pool.length > 1 && currentLine && nextLine.id === currentLine.id) {
    nextLine = pool.find(l => l.id !== currentLine.id) || nextLine;
  }

  currentLine = nextLine;
  displayLine(currentLine);

  // Trigger celebration effects for high scores
  if (window.confetti && currentLine.score >= 97) {
    confetti({
      particleCount: 40,
      spread: 60,
      origin: { y: 0.7 }
    });
  }
}

function displayLine(line) {
  const textEl = document.getElementById('pickup-text');
  const catTagEl = document.getElementById('card-cat-tag');
  const scoreNumEl = document.getElementById('card-rizz-num');
  const favIconSvg = document.getElementById('fav-icon-svg');

  // Fade out effect
  textEl.classList.add('opacity-0', 'scale-95');

  setTimeout(() => {
    textEl.innerText = `"${line.text}"`;

    const catLabels = {
      smooth: '🔥 Smooth & Flirty',
      brainrot: '🧠 Brainrot & Sigma',
      tech: '💻 Dev & IT Bro',
      vietnam: '☕ Thả Thính Tiếng Việt',
      cute: '🌸 Đáng Yêu',
      unhinged: '⚡ Táo Bạo & Chaos'
    };

    catTagEl.innerText = catLabels[line.cat] || line.cat.toUpperCase();
    scoreNumEl.innerText = `${line.score}%`;

    // Update favorite state
    const isFav = favoriteIds.includes(line.id);
    favIconSvg.setAttribute('fill', isFav ? 'currentColor' : 'none');
    favIconSvg.classList.toggle('text-amber-400', isFav);

    textEl.classList.remove('opacity-0', 'scale-95');
  }, 150);
}

// Copy to clipboard
function copyCurrentLine() {
  if (!currentLine) return;
  navigator.clipboard.writeText(currentLine.text).then(() => {
    playSound('copy');
    showToast('Đã copy câu thả thính vào clipboard! 📋');
  }).catch(() => {
    showToast('Lỗi copy, vui lòng thử lại!');
  });
}

// Text to speech narration
function speakCurrentLine() {
  if (!currentLine) return;
  playSound('click');
  if ('speechSynthesis' in window) {
    window.speechSynthesis.cancel();
    const utterance = new SpeechSynthesisUtterance(currentLine.text);
    utterance.lang = 'vi-VN';
    utterance.rate = 0.95;
    utterance.pitch = 1.05;
    window.speechSynthesis.speak(utterance);
    showToast('Đang phát giọng đọc Voice of Rizz... 🗣️');
  } else {
    showToast('Trình duyệt của bạn không hỗ trợ Text-to-Speech!');
  }
}

// Favorites toggle
function toggleFavoriteCurrent() {
  if (!currentLine) return;
  playSound('click');
  const index = favoriteIds.indexOf(currentLine.id);
  if (index > -1) {
    favoriteIds.splice(index, 1);
    showToast('Đã xóa khỏi danh sách yêu thích');
  } else {
    favoriteIds.push(currentLine.id);
    showToast('Đã lưu vào danh sách yêu thích! ⭐');
  }
  localStorage.setItem('rizzer_favs', JSON.stringify(favoriteIds));
  displayLine(currentLine);
  updateFavBadge();
}

function updateFavBadge() {
  const badge = document.getElementById('fav-badge');
  if (badge) badge.innerText = favoriteIds.length;
}

function renderFavorites() {
  const container = document.getElementById('favorites-grid');
  const favLines = RIZZ_DATABASE.filter(l => favoriteIds.includes(l.id));

  if (favLines.length === 0) {
    container.innerHTML = `
      <div class="col-span-2 text-center py-12 text-slate-500 bg-white/5 rounded-2xl border border-white/10">
        <p class="text-3xl mb-2">⭐</p>
        <p class="font-semibold text-slate-300">Chưa có câu thả thính nào được lưu</p>
        <p class="text-xs mt-1">Bấm biểu tượng ngôi sao ở trang "Tạo Rizz" để lưu nhé!</p>
      </div>
    `;
    return;
  }

  container.innerHTML = favLines.map(line => `
    <div class="rounded-2xl bg-white/5 border border-white/10 p-5 flex flex-col justify-between hover:border-pink-500/40 transition">
      <p class="text-slate-100 font-medium text-base mb-4 leading-snug">"${line.text}"</p>
      <div class="flex items-center justify-between pt-3 border-t border-white/10">
        <span class="text-xs text-pink-400 font-bold bg-pink-500/10 px-2 py-1 rounded">Rizz: ${line.score}%</span>
        <div class="flex items-center gap-2">
          <button onclick="copyText('${escapeQuotes(line.text)}')" class="p-2 rounded-lg bg-white/10 hover:bg-white/20 text-slate-300 text-xs flex items-center gap-1 cursor-pointer">
            Copy
          </button>
          <button onclick="removeFavorite(${line.id})" class="p-2 rounded-lg bg-red-500/20 hover:bg-red-500/30 text-red-400 text-xs cursor-pointer">
            Xóa
          </button>
        </div>
      </div>
    </div>
  `).join('');
}

function removeFavorite(id) {
  playSound('click');
  favoriteIds = favoriteIds.filter(favId => favId !== id);
  localStorage.setItem('rizzer_favs', JSON.stringify(favoriteIds));
  renderFavorites();
  updateFavBadge();
  showToast('Đã xóa khỏi yêu thích');
}

function clearAllFavorites() {
  if (favoriteIds.length === 0) return;
  playSound('click');
  favoriteIds = [];
  localStorage.setItem('rizzer_favs', JSON.stringify(favoriteIds));
  renderFavorites();
  updateFavBadge();
  showToast('Đã xóa toàn bộ yêu thích');
}

// Utility quote escape
function escapeQuotes(str) {
  return str.replace(/'/g, "\\'").replace(/"/g, '&quot;');
}

function copyText(txt) {
  navigator.clipboard.writeText(txt);
  playSound('copy');
  showToast('Đã sao chép! 📋');
}

// ================= RIZZOMETER AI ANALYZER =================
function analyzeRizzScore() {
  const input = document.getElementById('rizz-input').value.trim();
  if (!input) {
    showToast('Vui lòng nhập câu thả thính cần kiểm tra!');
    return;
  }

  playSound('magic');

  // Compute dynamic pseudo AI score based on text parameters
  let score = 50;
  const len = input.length;
  
  // Length factor
  if (len > 15 && len < 120) score += 20;
  else if (len >= 120) score += 10;

  // Keyword bonus
  const keywords = ['tim', 'yêu', 'mắt', 'cười', 'cậu', 'em', 'anh', 'rizz', 'sigma', 'bug', 'skibidi', 'fanum', 'say', 'đổ', 'đẹp'];
  keywords.forEach(kw => {
    if (input.toLowerCase().includes(kw)) score += 4;
  });

  // Hash determinism for consistent score per unique string
  let hash = 0;
  for (let i = 0; i < input.length; i++) {
    hash = (hash << 5) - hash + input.charCodeAt(i);
    hash |= 0;
  }
  score += Math.abs(hash % 20);

  // Clamp 45-100
  score = Math.min(100, Math.max(45, score));

  // Compute sub-breakdown
  const smoothVal = Math.min(100, score + (hash % 10));
  const creativeVal = Math.min(100, Math.max(50, score - (hash % 15)));
  const riskVal = Math.min(100, Math.max(30, (hash % 60) + 30));

  // Determine Rank & Verdict
  let rank = 'Casual Flirt';
  let verdict = 'Câu này khá ổn, có tiềm năng làm xiêu lòng đối phương!';
  let advice = 'Hãy giữ thái độ tự nhiên, không nên tỏ ra quá vội vã nhé.';

  if (score >= 95) {
    rank = '👑 Rizz God Supreme';
    verdict = '100% NO CAP! Câu thả thính này khiến Crush ngã ngửa vì quá mượt!';
    advice = 'Hãy chuẩn bị tinh thần nhận lời đồng ý ngay lập tức!';
  } else if (score >= 85) {
    rank = '⚡ Sigma Rizzer';
    verdict = 'Rizz level cực cao! Sát thương tình cảm gây ra là 9999 HP!';
    advice = 'Thêm 1 ánh mắt ngọt ngào nữa là chốt deal hoàn hảo.';
  } else if (score >= 70) {
    rank = '🔥 Smooth Operator';
    verdict = 'Rất tự nhiên và ấm áp. Đạt chuẩn thả thính tinh tế!';
    advice = 'Có thể mỉm cười nhẹ nhàng khi gửi câu này để tăng hiệu quả.';
  } else {
    rank = '🎯 Rookie Flirt';
    verdict = 'Cần luyện tập thêm, câu này hơi an toàn một chút.';
    advice = 'Thử dùng thêm bối cảnh hài hước hoặc chút gia vị Brainrot xem sao!';
  }

  // Animate Gauge & Update UI
  const resultBox = document.getElementById('rizzometer-result');
  resultBox.classList.remove('hidden');

  document.getElementById('res-badge').innerText = rank;
  document.getElementById('res-title').innerText = `${score} / 100 PTS`;
  document.getElementById('res-verdict').innerText = `"${verdict}"`;
  document.getElementById('res-score-num').innerText = score;
  document.getElementById('res-advice').innerText = advice;

  // Animate Progress Bars
  setTimeout(() => {
    document.getElementById('gauge-circle').setAttribute('stroke-dasharray', `${score}, 100`);

    document.getElementById('bar-smooth').style.width = `${smoothVal}%`;
    document.getElementById('bar-smooth-val').innerText = `${smoothVal}%`;

    document.getElementById('bar-creative').style.width = `${creativeVal}%`;
    document.getElementById('bar-creative-val').innerText = `${creativeVal}%`;

    document.getElementById('bar-risk').style.width = `${riskVal}%`;
    document.getElementById('bar-risk-val').innerText = `${riskVal}%`;
  }, 100);

  if (score >= 90 && window.confetti) {
    confetti({ particleCount: 50, spread: 70 });
  }
}

// ================= RIZZ CHAT SIMULATOR =================
const BOTS = {
  crush_cute: {
    name: 'Crush Đáng Yêu (Mai)',
    avatar: '🌸',
    welcome: 'Hì, chào cậu nha! Hôm nay trời đẹp thế, cậu đang làm gì đấy? ✨',
    responses: [
      { trigger: ['chào', 'hi', 'hello'], text: 'Héloo cậu! Nhìn cậu hôm nay vui tươi ghê đó 💖', delta: +5 },
      { trigger: ['đẹp', 'xinh', 'thích', 'yêu'], text: 'Ứ ừ... cậu nói vậy làm tui ngại muốn xỉu luôn nè 🙈', delta: +15 },
      { trigger: ['ăn', 'trà sữa', 'đi chơi', 'hẹn'], text: 'Dạ được chứ! Khi nào cậu rảnh thì dắt tui đi với nha 🧋', delta: +20 },
      { trigger: ['xấu', 'dở', 'chán'], text: 'Hừm, cậu nói thế là tui dỗi luôn đó nha! 🥺', delta: -10 }
    ],
    defaultReplies: [
      'Ui câu này nghe đáng yêu ghê á 🌸',
      'Thế hử? Kể tui nghe thêm đi!',
      'Hahaha cậu hài hước thật sự luôn đó! 😂'
    ]
  },
  crush_dev: {
    name: 'Crush Dev Girl (Linh)',
    avatar: '💻',
    welcome: '`git status` - Ready for incoming messages. Cậu tìm tui có việc gì thế?',
    responses: [
      { trigger: ['code', 'bug', 'git', 'api', 'dev'], text: 'Woah, cậu cũng biết dev hả? `Status 200 OK` luôn nè! 🚀', delta: +20 },
      { trigger: ['xinh', 'thích', 'yêu', 'merge'], text: 'Request hợp lệ! Anh chuẩn bị Merge Branch vào tim em chưa? 😉', delta: +15 },
      { trigger: ['cà phê', 'coffee', 'gặp'], text: 'Okie, để em `push` công việc xong rồi mình đi nhé!', delta: +10 }
    ],
    defaultReplies: [
      'Câu này syntax đúng đấy nhưng cần thêm chút cảm xúc nha!',
      'Hehe, em đang review code mà đọc tin nhắn này nụ cười bật sáng luôn!',
      'Nói tiếp đi anh, em đang lắng nghe nè 💻'
    ]
  },
  crush_meme: {
    name: 'Crush Mê Meme (Trang)',
    avatar: '🧠',
    welcome: 'Chào Sigma Rizzer! Hôm nay có Rizz gì mới mang ra đây xem nào? 🔥',
    responses: [
      { trigger: ['skibidi', 'fanum', 'sigma', 'rizz', 'gyatt'], text: 'No cap! Bro is cooking with maximum Rizz energy fr fr! 👑', delta: +25 },
      { trigger: ['thích', 'yêu', 'xinh'], text: 'Vãi, Rizz level này làm tui ngã ngửa không trượt phát nào! 😂', delta: +15 }
    ],
    defaultReplies: [
      'Thế là thế nào? Giải thích xem nào bro! 💀',
      'Được của ló đấy, Rizz 8.5/10 nha!',
      'Sigma mindset chuẩn đét luôn fr fr 🔥'
    ]
  }
};

function selectBot(botKey) {
  playSound('click');
  currentBot = botKey;
  chatRizzScore = 40;
  chatHistory = [];

  document.querySelectorAll('.bot-pill').forEach(btn => btn.classList.remove('active'));
  event.currentTarget.classList.add('active');

  const botData = BOTS[botKey];
  document.getElementById('chat-avatar').innerText = botData.avatar;
  document.getElementById('chat-bot-name').innerText = botData.name;

  // Add welcome message
  addChatMessage(botData.name, botData.welcome, 'bot');
  updateChatRizzBar(40);
}

function updateChatRizzBar(val) {
  chatRizzScore = Math.min(100, Math.max(0, val));
  document.getElementById('chat-rizz-bar').style.width = `${chatRizzScore}%`;
  document.getElementById('chat-rizz-val').innerText = `${chatRizzScore}%`;
}

function addChatMessage(sender, text, type) {
  const container = document.getElementById('chat-messages');
  const msgDiv = document.createElement('div');
  msgDiv.className = `flex gap-2.5 ${type === 'user' ? 'justify-end' : 'justify-start'} animate-fadeIn`;

  const isUser = type === 'user';
  msgDiv.innerHTML = `
    ${!isUser ? `<div class="w-8 h-8 rounded-full bg-pink-500/20 flex items-center justify-center text-sm border border-pink-500/30 flex-shrink-0">${BOTS[currentBot].avatar}</div>` : ''}
    <div class="max-w-[78%] rounded-2xl px-4 py-2.5 text-sm ${isUser ? 'bg-gradient-to-r from-pink-600 to-purple-600 text-white rounded-tr-none shadow-md' : 'bg-white/10 text-slate-100 border border-white/10 rounded-tl-none'}">
      <p class="text-[11px] font-semibold opacity-70 mb-0.5">${sender}</p>
      <p class="leading-relaxed">${text}</p>
    </div>
  `;

  container.appendChild(msgDiv);
  container.scrollTop = container.scrollHeight;
}

function sendChatMessage() {
  const input = document.getElementById('chat-user-input');
  const text = input.value.trim();
  if (!text) return;

  playSound('click');
  addChatMessage('Bạn', text, 'user');
  input.value = '';

  // Calculate bot response
  const botData = BOTS[currentBot];
  let matched = null;

  for (const resp of botData.responses) {
    if (resp.trigger.some(trig => text.toLowerCase().includes(trig))) {
      matched = resp;
      break;
    }
  }

  setTimeout(() => {
    let replyText = '';
    let delta = +5;

    if (matched) {
      replyText = matched.text;
      delta = matched.delta;
    } else {
      replyText = botData.defaultReplies[Math.floor(Math.random() * botData.defaultReplies.length)];
    }

    playSound('success');
    addChatMessage(botData.name, replyText, 'bot');
    updateChatRizzBar(chatRizzScore + delta);

    if (chatRizzScore >= 90 && window.confetti) {
      confetti({ particleCount: 35, spread: 50 });
    }
  }, 700);
}

// ================= RIZZ CARD CANVAS EXPORTER =================
function openCardExportModal() {
  if (!currentLine) return;
  playSound('click');
  document.getElementById('card-modal').classList.remove('hidden');
  document.getElementById('card-modal').classList.add('flex');
  renderRizzCanvas();
}

function closeCardExportModal() {
  playSound('click');
  document.getElementById('card-modal').classList.add('hidden');
  document.getElementById('card-modal').classList.remove('flex');
}

function setCanvasTheme(theme) {
  playSound('click');
  canvasTheme = theme;
  renderRizzCanvas();
}

function renderRizzCanvas() {
  const canvas = document.getElementById('rizz-canvas');
  if (!canvas || !currentLine) return;

  const ctx = canvas.getContext('2d');
  const width = canvas.width;
  const height = canvas.height;

  // Background Gradient based on Theme
  let grad = ctx.createLinearGradient(0, 0, width, height);
  if (canvasTheme === 'sunset') {
    grad.addColorStop(0, '#f97316');
    grad.addColorStop(0.5, '#ec4899');
    grad.addColorStop(1, '#8b5cf6');
  } else if (canvasTheme === 'cyan') {
    grad.addColorStop(0, '#0f172a');
    grad.addColorStop(0.5, '#0284c7');
    grad.addColorStop(1, '#06b6d4');
  } else if (canvasTheme === 'darkgold') {
    grad.addColorStop(0, '#1e1b4b');
    grad.addColorStop(0.5, '#581c87');
    grad.addColorStop(1, '#d97706');
  } else {
    // Cyber Neon (Default)
    grad.addColorStop(0, '#0d0722');
    grad.addColorStop(0.5, '#be185d');
    grad.addColorStop(1, '#4c1d95');
  }

  ctx.fillStyle = grad;
  ctx.fillRect(0, 0, width, height);

  // Decorative ambient circles
  ctx.fillStyle = 'rgba(255, 255, 255, 0.08)';
  ctx.beginPath();
  ctx.arc(width - 40, 40, 120, 0, Math.PI * 2);
  ctx.fill();

  ctx.beginPath();
  ctx.arc(40, height - 40, 90, 0, Math.PI * 2);
  ctx.fill();

  // Glass Card Inner Frame
  ctx.fillStyle = 'rgba(15, 10, 30, 0.55)';
  ctx.strokeStyle = 'rgba(255, 255, 255, 0.25)';
  ctx.lineWidth = 2;
  roundRect(ctx, 25, 25, width - 50, height - 50, 20, true, true);

  // Header Title & Logo
  ctx.font = 'bold 18px Outfit, sans-serif';
  ctx.fillStyle = '#ff007f';
  ctx.fillText('RIZZER PRO MAX ⚡', 50, 65);

  ctx.font = '12px Outfit, sans-serif';
  ctx.fillStyle = 'rgba(255, 255, 255, 0.7)';
  ctx.fillText(`Rizz Power: ${currentLine.score}%`, width - 180, 65);

  // Divider Line
  ctx.strokeStyle = 'rgba(255, 255, 255, 0.15)';
  ctx.lineWidth = 1;
  ctx.beginPath();
  ctx.moveTo(50, 80);
  ctx.lineTo(width - 50, 80);
  ctx.stroke();

  // Pickup Text wrapped inside Canvas
  ctx.font = 'bold 22px Outfit, Poppins, sans-serif';
  ctx.fillStyle = '#ffffff';

  const text = `"${currentLine.text}"`;
  wrapText(ctx, text, 50, 140, width - 100, 34);

  // Footer Tag
  ctx.font = '12px Outfit, sans-serif';
  ctx.fillStyle = 'rgba(255, 255, 255, 0.5)';
  ctx.fillText('Generated by Rizzer Web App • 2026 Edition', 50, height - 45);
}

// Canvas Utility functions
function roundRect(ctx, x, y, width, height, radius, fill, stroke) {
  ctx.beginPath();
  ctx.moveTo(x + radius, y);
  ctx.lineTo(x + width - radius, y);
  ctx.quadraticCurveTo(x + width, y, x + width, y + radius);
  ctx.lineTo(x + width, y + height - radius);
  ctx.quadraticCurveTo(x + width, y + height, x + width - radius, y + height);
  ctx.lineTo(x + radius, y + height);
  ctx.quadraticCurveTo(x, y + height, x, y + height - radius);
  ctx.lineTo(x, y + radius);
  ctx.quadraticCurveTo(x, y, x + radius, y);
  ctx.closePath();
  if (fill) ctx.fill();
  if (stroke) ctx.stroke();
}

function wrapText(ctx, text, x, y, maxWidth, lineHeight) {
  const words = text.split(' ');
  let line = '';

  for (let n = 0; n < words.length; n++) {
    const testLine = line + words[n] + ' ';
    const metrics = ctx.measureText(testLine);
    const testWidth = metrics.width;
    if (testWidth > maxWidth && n > 0) {
      ctx.fillText(line, x, y);
      line = words[n] + ' ';
      y += lineHeight;
    } else {
      line = testLine;
    }
  }
  ctx.fillText(line, x, y);
}

function downloadCanvasImage() {
  const canvas = document.getElementById('rizz-canvas');
  if (!canvas) return;
  playSound('success');
  const link = document.createElement('a');
  link.download = `rizz-card-${Date.now()}.png`;
  link.href = canvas.toDataURL('image/png');
  link.click();
  showToast('Đã tải ảnh Rizz Card về máy! 🖼️');
}

function copyCanvasImageToClipboard() {
  const canvas = document.getElementById('rizz-canvas');
  if (!canvas) return;
  canvas.toBlob(blob => {
    try {
      const item = new ClipboardItem({ 'image/png': blob });
      navigator.clipboard.write([item]);
      playSound('copy');
      showToast('Đã copy ảnh Card vào bộ nhớ tạm! 📋');
    } catch (e) {
      showToast('Trình duyệt chưa hỗ trợ copy ảnh trực tiếp, hãy dùng Tải Ảnh.');
    }
  });
}

// ================= TOAST NOTIFICATION SYSTEM =================
function showToast(msg) {
  const container = document.getElementById('toast-container');
  if (!container) return;

  const toast = document.createElement('div');
  toast.className = 'px-4 py-3 rounded-2xl bg-[#1d173b] border border-pink-500/40 text-slate-100 font-medium text-xs sm:text-sm shadow-xl flex items-center gap-2 animate-fadeIn pointer-events-auto backdrop-blur-xl';
  toast.innerHTML = `
    <span class="w-2 h-2 rounded-full bg-pink-400"></span>
    <span>${msg}</span>
  `;

  container.appendChild(toast);
  setTimeout(() => {
    toast.classList.add('opacity-0', 'transition', 'duration-300');
    setTimeout(() => toast.remove(), 300);
  }, 2600);
}

// ================= INITIALIZATION =================
window.addEventListener('DOMContentLoaded', () => {
  // Update total lines count stat
  const statEl = document.getElementById('stat-total-lines');
  if (statEl) statEl.innerText = RIZZ_DATABASE.length;

  updateFavBadge();
  generateRizz();

  // Initialize bot chat greeting
  selectBot('crush_cute');
});
