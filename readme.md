A turn based tactics multiplayer game with destructible terrain and free aim.

# In one sentence

It takes the general idea of XCOM, adds back the ballistics system from the original
spiked with the free aim mode taken from Phoenix Point and puts it into a minecraft style
destructible block world, with a focus on multiplayer matches.

# Motivation

I really like X-Com and XCOM. I've played these games a lot.

While the original is cool, the balancing is lacking.

The Firaxis reboot is great, but I don't like the "hit chance" system.

When Phoenix Point came along and introduced us to the combination
of XCOM style combat with the ability to freely aim at anything from
a first person perspective, I was hooked. To me this, idea is really genius:
Adding even a basic ballistics system seems like an ideal fit.

Funny enough, the original X-Com from the 90s was more realistic in this regard, than
the Firaxis ones:
-> https://www.ufopaedia.org/index.php/LOFTEMPS.DAT

Phoenix Point is a great game, but it has some flaws. It's again the balancing.

I am also missing a good multiplayer mode. Phoenix Point has none, and the one
in XCOM is more an afterthought. So I wrote this game engine as client/server
architecture from the start. When running a standalone singleplayer game,
it will actually spawn a server instance and a headless client in separate threads.

We already had most of this: https://ufo2000.sourceforge.net/

So, basically I started out with this project because I wanted to have a game that
has these characteristics:

- 3D XCOM style turn based combat
- Phoenix Point style "free aiming"
- a basic ballistics system against character models
- built for classic client/server multiplayer
- Destruction of terrain and objects
- Easy editing & modding support from the start

I felt that going with a minecraft style pseudo-voxel block engine would be a good fit.
It has destruction by definition and allows for easy editing.

It also allows this project to use the big amount of tools & resources available for
minecraft modding.

## Gameplay 101

You can move during free aim mode within your current block.

Sniper rifles need concentration to be accurate. Visible enemy units near your units will put
pressure on them, making it harder to concentrate, thus reducing accuracy.

Overwatch will always trigger when an enemy steps on that block.
It incurs a 20% accuracy penalty for the reaction but adds a 10% damage modifier,
since the enemy is unprepared.