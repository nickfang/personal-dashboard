<script lang="ts">
  export let data;
  let activeItem = 'arc';
  let setActiveItem = (name: string) => {
    activeItem = name.toLowerCase();
  };
</script>

<main>
  <div>
    <div class="banner">
      <h1>Geometry</h1>
      <p>
        Geometry is the branch of mathematics that deals with the properties and relationships of
        points, lines, angles, surfaces, and solids. It is the study of shapes and figures and how
        they interact with each other. Geometry is an important field of mathematics that has many
        practical applications in everyday life, such as in architecture, engineering, and art.
      </p>
    </div>
    <div class="container">
      <div class="sidebar">
        {#each Object.entries(data.Shapes) as [key, shape]}
          <button on:click={() => setActiveItem(shape.name)}>{shape.name}</button>
        {/each}
      </div>
      <div class="content">
        <div class="content-item">
          {#if activeItem}
            <h2>{data.Shapes[activeItem].name}</h2>
            <div>{data.Shapes[activeItem].description}</div>
            <div>Equations</div>
            {#each Object.entries(data.Shapes[activeItem].equations) as [key, equation]}
              <div>{key}: {equation}</div>
            {/each}
          {/if}
        </div>
      </div>
    </div>
  </div>
</main>

<style>
  .banner {
    width: 50vw;
    margin: 16px auto;
    padding: 64px;
  }

  .container {
    display: flex;
    flex-direction: row;

    gap: 2rem;
  }
  .sidebar {
    width: 15vw;
    margin: 0 32px;
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }
  .content {
    width: 30vw;
    display: flex;
    justify-content: center;
    align-items: center;
    border: black 1px solid;
    padding: 20rem;
    margin-left: 2rem;
  }
  .content-item {
    padding-bottom: 64px;
  }
  button {
    padding: 0.75rem 1rem;
    border: none;
    background-color: #f0f0f0;
    cursor: pointer;
  }
</style>
