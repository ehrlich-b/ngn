# ngn - Chess Engine

ngn is a high-performance chess engine written in Go.

## Current Status: 🎉 **BOTH MAJOR CRISES RESOLVED!** 🎉

✅ **Performance Crisis RESOLVED**: 120,000+ NPS (exceeded 100K red line)  
✅ **Evaluation Crisis RESOLVED**: 1477 ELO (exceeded 1400 red line)

**Ready for advanced feature development with solid foundation!**

## Quick Start

```bash
make build    # Build the engine
make test     # Run tests with performance monitoring  
make eval     # Run comprehensive 2-minute evaluation (occasional use)
```

## Development

- **Architecture**: Modular Go codebase with bitboard representation
- **Performance**: 120K+ NPS with zero-allocation hot paths
- **Strength**: 1477 ELO with proper time allocation
- **Interface**: Full UCI protocol support for chess GUIs

See `CLAUDE.md` for detailed development guidelines and `TODO.md` for current roadmap.
