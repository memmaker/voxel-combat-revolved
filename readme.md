# Motivation

I really like X-Com and XCOM. I've played these games a lot.

But I cannot stand the completely random chance to hit anymore.

When Phoenix Point came along and introduced us to the combination
of XCOM style combat with the ability to freely aim at anything from
a first person perspective, I was hooked. To me this, idea is really genius:
Adding even a basic ballistics system seems like an ideal fit.

Funny enough, the original X-Com from the 90s was more realistic in this regard.

I am also missing a good multiplayer mode. Phoenix Point has none, and the one
in XCOM is more an afterthought. So I wrote this game engine as client/server
architecture from the start. When running a standalone singleplayer game,
it will actually spawn a server instance and a headless client in separate threads.

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