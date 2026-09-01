let sent = 0;
let recv = 0;
module.exports = {
  nextMessage: (ctx, ee, next) => { ctx.vars.msgSeq = (ctx.vars.msgSeq || 0) + 1; return next(); },
  trackSend: (ctx, ee, next) => { sent++; ee.emit('counter', 'soak.sent', 1); return next(); },
  trackRecv: (ctx, ee, next) => {
    recv++;
    return next();
  },
  afterResponse: (req, res, ctx, ee, next) => { return next(); },
  beforeScenario: (ctx, ee, next) => next(),
  afterScenario: (ctx, ee, next) => next(),
};
